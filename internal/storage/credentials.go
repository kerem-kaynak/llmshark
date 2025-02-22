package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"os"
)

type Credentials struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

type CredentialStore struct {
	path string
	key  []byte
}

func NewCredentialStore(path string) (*CredentialStore, error) {
	keyPath := path + ".key"
	key := make([]byte, 32)

	if _, err := os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
		// Generate new key
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, err
		}
		if err := os.WriteFile(keyPath, key, 0600); err != nil {
			return nil, err
		}
	} else {
		// Read existing key
		key, err = os.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
	}

	return &CredentialStore{
		path: path,
		key:  key,
	}, nil
}

func (s *CredentialStore) Save(creds *Credentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return os.WriteFile(s.path, ciphertext, 0600)
}

func (s *CredentialStore) Load() (*Credentials, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil // No credentials stored yet
		}
		return nil, err
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	var creds Credentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}
