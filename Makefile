# Makefile

BINARY_NAME=llmshark
VERSION?=1.0.0
GOPATH=$(shell go env GOPATH)
INSTALL_PATH=$(GOPATH)/bin

# Build settings
LDFLAGS=-ldflags "-X main.Version=${VERSION}"
GOARCH=$(shell go env GOARCH)
GOOS=$(shell go env GOOS)

.PHONY: all build clean install uninstall

all: build

build:
	@echo "Building ${BINARY_NAME}..."
	@go build ${LDFLAGS} -o ${BINARY_NAME} cmd/main.go

install: build
	@echo "Installing ${BINARY_NAME} to ${INSTALL_PATH}..."
	@cp ${BINARY_NAME} ${INSTALL_PATH}/${BINARY_NAME}
	@echo "Running installation script..."
	@chmod +x install.sh
	@PATH=$(PATH):$(GOPATH)/bin ./install.sh

uninstall:
	@echo "Removing ${BINARY_NAME}..."
	@rm -f ${INSTALL_PATH}/${BINARY_NAME}

clean:
	@echo "Cleaning..."
	@rm -f ${BINARY_NAME}
	@rm -f ${INSTALL_PATH}/${BINARY_NAME}
	@go clean
