BIN_DIR := bin
APP_NAME := kdeploy
BINARY := $(BIN_DIR)/$(APP_NAME)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

GOFILES := $(shell find . -name "*.go" -not -path "./.beads/*")

DEMO_TAPES := \
	demo/help.tape \
	demo/check.tape \
	demo/read-only.tape \
	demo/first-run.tape \
	demo/deploy.tape \
	demo/deploy-80.tape \
	demo/download.tape \
	demo/download-80.tape \
	demo/destroy.tape \
	demo/destroy-80.tape

.PHONY: all build format fmt-check vet lint test test-race check clean help demo

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

# Render only the safe tapes. The full-deploy tape, if added locally, is never
# part of this target and must not run in CI.
demo: build
	@for tool in vhs ttyd ffmpeg; do \
		command -v "$$tool" >/dev/null 2>&1 || { \
			echo "demo: missing $$tool (install vhs, ttyd, and ffmpeg)"; \
			exit 1; \
		}; \
	done
	@mkdir -p docs/media
	@for tape in $(DEMO_TAPES); do \
		echo "Rendering $$tape..."; \
		vhs "$$tape" || exit 1; \
		case "$$tape" in \
			demo/deploy*) \
				if [ -s /var/tmp/kdeploy-demo/runtime-operations.log ]; then \
					echo "demo: $$tape attempted a VM operation; refusing to continue"; \
					exit 1; \
				fi ;; \
			demo/destroy*) \
				if [ -s /var/tmp/kdeploy-demo/virsh-operations.log ]; then \
					echo "demo: $$tape attempted a virsh operation; refusing to continue"; \
					exit 1; \
				fi ;; \
		esac; \
	done
	@if [ -s /var/tmp/kdeploy-demo/runtime-operations.log ] || [ -s /var/tmp/kdeploy-demo/virsh-operations.log ]; then \
		echo "demo: unsafe VM operation detected; refusing to finish"; \
		exit 1; \
	fi

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
	@echo "  demo       - Render the safe VHS tapes into docs/media (dev-only)"
	@echo "  clean      - Remove build artifacts"
