.PHONY: help build build-fast clean test test-coverage test-bench fmt vet lint tidy migrate serve serve-ssh import-full import-updates list-sets install deps all check run

BINARY_NAME=cardman
BINARY_EXT=.exe
BUILD_DIR=.
CMD_DIR=./cmd
MAIN_FILE=$(CMD_DIR)/main.go

export CGO_ENABLED=1

help:
	@echo "tui-cardman - Terminal UI Card Manager"
	@echo ""
	@echo "Available targets:"
	@echo "  make build          - Build the application (CGO_ENABLED=1)"
	@echo "  make build-fast     - Fast build without optimizations"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make test           - Run all tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make test-bench     - Run all benchmarks"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make lint           - Run linter (golangci-lint)"
	@echo "  make tidy           - Tidy dependencies"
	@echo "  make check          - Run fmt, vet, and test"
	@echo "  make all            - Build, test, and check everything"
	@echo "  make migrate        - Run database migrations"
	@echo "  make serve          - Start TUI server"
	@echo "  make serve-ssh      - Start SSH server"
	@echo "  make import-full    - Import all Pokemon TCG data"
	@echo "  make import-updates - Import new Pokemon TCG sets only"
	@echo "  make list-sets      - List available Pokemon TCG sets"
	@echo "  make install        - Install binary to GOPATH/bin"
	@echo "  make deps           - Download dependencies"
	@echo "  make run            - Build and run (serve)"

build: $(BINARY_NAME)$(BINARY_EXT)

$(BINARY_NAME)$(BINARY_EXT):
	go build -o $(BINARY_NAME)$(BINARY_EXT) $(MAIN_FILE)

build-fast:
	go build -o $(BINARY_NAME)$(BINARY_EXT) $(CMD_DIR)

clean:
	@if exist $(BINARY_NAME)$(BINARY_EXT) del /Q $(BINARY_NAME)$(BINARY_EXT)
	@echo Cleaned build artifacts

test:
	go test ./...

test-coverage:
	go test -cover ./...

test-bench:
	go test -bench=. ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	@where golangci-lint >nul 2>&1 && golangci-lint run || echo golangci-lint not found, skipping

tidy:
	go mod tidy

deps:
	go mod download

check: fmt vet test

all: tidy check build

migrate: build
	.\$(BINARY_NAME)$(BINARY_EXT) migrate

serve: build
	.\$(BINARY_NAME)$(BINARY_EXT) serve

serve-ssh: build
	.\$(BINARY_NAME)$(BINARY_EXT) serve-ssh

import-full: build
	.\$(BINARY_NAME)$(BINARY_EXT) import-full

import-updates: build
	.\$(BINARY_NAME)$(BINARY_EXT) import-updates

list-sets: build
	.\$(BINARY_NAME)$(BINARY_EXT) list-sets

install:
	go install $(CMD_DIR)

run: build serve
