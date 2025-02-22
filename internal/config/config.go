package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	CredentialsPath string
}

func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".llmshark")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, err
	}

	return &Config{
		CredentialsPath: filepath.Join(configDir, "credentials.enc"),
	}, nil
}
