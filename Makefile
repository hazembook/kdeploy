BIN_DIR := bin
APP_NAME := kdeploy
BINARY := $(BIN_DIR)/$(APP_NAME)

GOFILES := $(shell find . -name "*.go" -not -path "./.beads/*")

.PHONY: all build format vet lint test clean help

all: build

build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) .

format:
	@echo "Formatting code..."
	gofmt -s -w .
	goimports -w .

vet:
	@echo "Running go vet..."
	go vet ./...

test:
	@echo "Running tests..."
	go test ./... -v

lint: format vet
	@echo "Running golangci-lint..."
	golangci-lint run

clean:
	@echo "Cleaning up..."
	rm -rf $(BIN_DIR)

help:
	@echo "Available targets:"
	@echo "  build   - Build the kdeploy binary into bin/"
	@echo "  format  - Format code using gofmt and goimports"
	@echo "  vet     - Run go vet"
	@echo "  test    - Run all tests with verbose output"
	@echo "  lint    - Run format, vet, and golangci-lint"
	@echo "  clean   - Remove build artifacts"
