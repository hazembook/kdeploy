BIN_DIR := bin
APP_NAME := kdeploy
BINARY := $(BIN_DIR)/$(APP_NAME)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

GOFILES := $(shell find . -name "*.go" -not -path "./.beads/*")

.PHONY: all build format fmt-check vet lint test test-race check clean help

all: build

build:
	@echo "Building $(BINARY) ($(VERSION))..."
	@mkdir -p $(BIN_DIR)
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

format:
	@echo "Formatting code..."
	gofmt -s -w .
	goimports -w .

fmt-check:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted Go files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

vet:
	@echo "Running go vet..."
	go vet ./...

test:
	@echo "Running tests..."
	go test ./... -v

test-race:
	@echo "Running race tests..."
	go test -race ./...

check: fmt-check vet test-race

lint: format vet
	@echo "Running golangci-lint..."
	golangci-lint run

clean:
	@echo "Cleaning up..."
	rm -rf $(BIN_DIR)

help:
	@echo "Available targets:"
	@echo "  build      - Build the version-injected kdeploy binary into bin/"
	@echo "  format     - Format code using gofmt and goimports"
	@echo "  fmt-check  - Fail if any Go file is not gofmt-clean (no writes)"
	@echo "  vet        - Run go vet"
	@echo "  test       - Run all tests with verbose output"
	@echo "  test-race  - Run all tests with the race detector"
	@echo "  check      - fmt-check + vet + test-race (the pre-commit gate)"
	@echo "  lint       - Run format, vet, and golangci-lint"
	@echo "  clean      - Remove build artifacts"
