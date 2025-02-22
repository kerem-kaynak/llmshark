#!/bin/bash

# Get the user's home directory
HOME_DIR="$HOME"

# Determine the Go binary directory
GO_BIN_DIR=$(go env GOPATH)/bin

# Determine the shell being used
SHELL_NAME=$(basename "$SHELL")

# Function to add to PATH
add_to_path() {
  if grep -q "$GO_BIN_DIR" "$1"; then
    echo "$GO_BIN_DIR is already in your PATH in $1."
  else
    echo "Adding $GO_BIN_DIR to PATH in $1"
    echo "export PATH=\"\$PATH:$GO_BIN_DIR\"" >> "$1"
  fi
}

# Handle different shell types
if [[ "$SHELL_NAME" == "bash" ]]; then
  CONFIG_FILE="$HOME_DIR/.bashrc"
  add_to_path "$CONFIG_FILE"
elif [[ "$SHELL_NAME" == "zsh" ]]; then
  CONFIG_FILE="$HOME_DIR/.zshrc"
  add_to_path "$CONFIG_FILE"
else
  echo "Unsupported shell: $SHELL_NAME. Please manually add $GO_BIN_DIR to your PATH."
  exit 1
fi

# Print instructions to update the current session
echo "LLMShark has been successfully installed."
echo "Please run 'source $CONFIG_FILE' or open a new terminal to start using llmshark."
