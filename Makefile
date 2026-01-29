.PHONY: help build build-fast clean clean-all test test-coverage test-bench fmt vet lint lint-install tidy migrate serve serve-ssh import-full import-updates list-sets install deps all check run

BINARY_NAME=cardman
BUILD_DIR=.
CMD_DIR=./cmd
MAIN_FILE=$(CMD_DIR)/main.go

ifeq ($(OS),Windows_NT)
	BINARY_EXT=.exe
	RUNBIN=.\\$(BINARY_NAME)$(BINARY_EXT)
	RM_CMD=del /Q
	NULLDEV=NUL
else
	BINARY_EXT=
	RUNBIN=./$(BINARY_NAME)$(BINARY_EXT)
	RM_CMD=rm -f
	NULLDEV=/dev/null
endif

export CGO_ENABLED=1

help:
	@echo "tui-cardman - Terminal UI Card Manager"
	@echo ""
	@echo "Available targets:"
	@echo "  make build          - Build the application (CGO_ENABLED=1)"
	@echo "  make build-fast     - Fast build without optimizations"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make clean-all      - Remove build artifacts, database, and logs"
	@echo "  make test           - Run all tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make test-bench     - Run all benchmarks"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make lint           - Run linter (golangci-lint)"
	@echo "  make lint-install   - Install golangci-lint"
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

ifeq ($(OS),Windows_NT)
clean:
	-@if exist $(BINARY_NAME)$(BINARY_EXT) $(RM_CMD) $(BINARY_NAME)$(BINARY_EXT)
	@echo Cleaned build artifacts

clean-all: clean
	-@if exist cardman.db $(RM_CMD) cardman.db
	-@if exist output.log $(RM_CMD) output.log
	@echo Cleaned all artifacts, database, and logs
else
clean:
	-@$(RM_CMD) $(BINARY_NAME)$(BINARY_EXT)
	@echo Cleaned build artifacts

clean-all: clean
	-@$(RM_CMD) cardman.db output.log
	@echo Cleaned all artifacts, database, and logs
endif

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
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo golangci-lint not found, skipping

lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

tidy:
	go mod tidy

deps:
	go mod download

check: fmt vet test

all: tidy check build


migrate: build
	$(RUNBIN) migrate

serve: build
	$(RUNBIN) serve

serve-ssh: build
	$(RUNBIN) serve-ssh

import-full: build
	$(RUNBIN) import-full

import-updates: build
	$(RUNBIN) import-updates

list-sets: build
	$(RUNBIN) list-sets

install:
	go install $(CMD_DIR)

run: build serve
