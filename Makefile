APP_NAME := scalable-ecommerce-platform
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.CommitHash=$(COMMIT_HASH)"
MAIN_PATH := ./cmd/$(APP_NAME)
BIN_DIR := ./bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
SANITIZED_VERSION := $(shell echo $(VERSION) | sed 's/[^a-zA-Z0-9.-]/-/g')
DOCKER_IMAGE := $(APP_NAME):$(SANITIZED_VERSION)
DOCKER_LATEST := $(APP_NAME):latest
GO_FILES := $(shell find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./.go/*")
GO_PACKAGES := $(shell go list ./... | grep -v /vendor/)
COVER_PACKAGES := $(shell go list ./... | grep -vE 'mocks|docs|testutils|cmd|internal/utils|internal/errors|internal/health|internal/metrics|pkg/stripe|pkg/sendGrid/mocks')
GO_BUILD_ENV := CGO_ENABLED=0
GO_VERSION := $(shell go version | cut -d " " -f 3 | cut -d "o" -f 2)
REQUIRED_GO_VERSION := 1.22  # Update to minimum required Go version

# Define the architecture to build for
ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
	GOARCH := amd64
else ifeq ($(ARCH),arm64)
	GOARCH := arm64
else
	GOARCH := $(ARCH)
endif

# Improve OS detection
ifeq ($(OS),Windows_NT)
	DETECTED_OS := windows
	BINARY_EXT := .exe
	OPEN_CMD := start
	RM_CMD := del /F /Q
	RMDIR_CMD := rmdir /S /Q
	MKDIR_CMD := if not exist "$(subst /,\\,$(1))" mkdir "$(subst /,\\,$(1))"
else
	DETECTED_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
	BINARY_EXT :=
	ifeq ($(DETECTED_OS),darwin)
		OPEN_CMD := open
	else
		OPEN_CMD := $(shell command -v xdg-open >/dev/null 2>&1 && echo "xdg-open" || echo "true")
	endif
	RM_CMD := rm -f
	RMDIR_CMD := rm -rf
	MKDIR_CMD := mkdir -p
endif

# Output control
ifeq ($(VERBOSE),true)
	ECHO := @true
else
	ECHO := @echo
endif

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
RED := \033[0;31m
NC := \033[0m # No Color

# Docker configuration
DOCKER_BUILDKIT ?= 1
DOCKER_BUILD_ARGS ?= --build-arg APP_VERSION=$(VERSION)
DOCKER_REGISTRY ?= 
DOCKER_FULL_IMAGE := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY)/$(DOCKER_IMAGE),$(DOCKER_IMAGE))
DOCKER_FULL_LATEST := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY)/$(DOCKER_LATEST),$(DOCKER_LATEST))

# Check if Go version meets requirements
.PHONY: check-go-version
check-go-version: ## Check if Go version meets requirements
	@echo "Checking Go version..."
	@if [ "$(GO_VERSION)" \< "$(REQUIRED_GO_VERSION)" ]; then \
		echo "$(RED)Error: Go version $(GO_VERSION) is less than required version $(REQUIRED_GO_VERSION)$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Go version $(GO_VERSION) is acceptable$(NC)"

.PHONY: all
all: clean deps fmt lint vet test build ## Run clean, deps, fmt, lint, vet, test, and build

.PHONY: build
build: check-go-version ## Build the application
	$(ECHO) "$(BLUE)Building $(APP_NAME) version $(VERSION)...$(NC)"
	@$(MKDIR_CMD) $(BIN_DIR)
	@$(GO_BUILD_ENV) GOOS=$(DETECTED_OS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BIN_PATH)$(BINARY_EXT) $(MAIN_PATH)
	$(ECHO) "$(GREEN)Build complete: $(BIN_PATH)$(BINARY_EXT)$(NC)"

# Build for multiple platforms
.PHONY: build-all
build-all: check-go-version build-linux build-mac build-windows ## Build for all platforms (Linux, macOS, Windows)

.PHONY: build-linux
build-linux: ## Build for Linux platforms
	$(ECHO) "$(BLUE)Building for Linux...$(NC)"
	@$(MKDIR_CMD) $(BIN_DIR)/linux
	@$(GO_BUILD_ENV) GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/linux/$(APP_NAME) $(MAIN_PATH)
	@$(GO_BUILD_ENV) GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/linux/$(APP_NAME)-arm64 $(MAIN_PATH)
	$(ECHO) "$(GREEN)Linux builds complete$(NC)"

.PHONY: build-mac
build-mac: ## Build for macOS platforms
	$(ECHO) "$(BLUE)Building for macOS...$(NC)"
	@$(MKDIR_CMD) $(BIN_DIR)/darwin
	@$(GO_BUILD_ENV) GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/darwin/$(APP_NAME) $(MAIN_PATH)
	@$(GO_BUILD_ENV) GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/darwin/$(APP_NAME)-arm64 $(MAIN_PATH)
	$(ECHO) "$(GREEN)macOS builds complete$(NC)"

.PHONY: build-windows
build-windows: ## Build for Windows platform
	$(ECHO) "$(BLUE)Building for Windows...$(NC)"
	@$(MKDIR_CMD) $(BIN_DIR)/windows
	@$(GO_BUILD_ENV) GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/windows/$(APP_NAME).exe $(MAIN_PATH)
	$(ECHO) "$(GREEN)Windows build complete$(NC)"

.PHONY: run
run: build ## Run the application
	$(ECHO) "$(BLUE)Running $(APP_NAME)...$(NC)"
	@$(BIN_PATH)$(BINARY_EXT)

.PHONY: test
test: ## Run tests with race detection
	$(ECHO) "$(BLUE)Running tests with race detection...$(NC)"
	@go test -race -cover -parallel=8 ./...

.PHONY: test-coverage
test-coverage: ## Generate test coverage report
	$(ECHO) "$(BLUE)Generating test coverage report...$(NC)"
	@go test -coverprofile=coverage.out $(COVER_PACKAGES)
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	$(ECHO) "$(GREEN)Coverage report generated: coverage.html$(NC)"
	@$(OPEN_CMD) coverage.html

.PHONY: test-integration
test-integration: ## Run integration tests
	$(ECHO) "$(BLUE)Running integration tests...$(NC)"
	@go test -tags=integration -parallel=4 ./...

.PHONY: lint
lint: ## Run linter
	$(ECHO) "$(BLUE)Running linter...$(NC)"
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "$(YELLOW)golangci-lint not found, installing...$(NC)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest || { echo "$(RED)Failed to install golangci-lint$(NC)"; exit 1; }; \
	fi
	@golangci-lint run --timeout=5m

.PHONY: fmt
fmt: ## Format code
	$(ECHO) "$(BLUE)Formatting code...$(NC)"
	@gofmt -s -w $(GO_FILES)

.PHONY: vet
vet: ## Run go vet
	$(ECHO) "$(BLUE)Running go vet...$(NC)"
	@go vet $(GO_PACKAGES)

.PHONY: deps
deps: check-go-version ## Verify and update dependencies
	$(ECHO) "$(BLUE)Verifying and updating dependencies...$(NC)"
	@go mod tidy
	@go mod verify

.PHONY: clean
clean: ## Clean build artifacts
	$(ECHO) "$(BLUE)Cleaning up...$(NC)"
	@$(RM_CMD) $(BIN_PATH)$(BINARY_EXT) 2>/dev/null || true
	@$(RM_CMD) coverage.out filtered_coverage.out coverage.html 2>/dev/null || true
	@$(RMDIR_CMD) $(BIN_DIR) 2>/dev/null || true
	@go clean -cache -testcache

.PHONY: docker-build
docker-build: ## Build Docker image
	$(ECHO) "$(BLUE)Building Docker image $(DOCKER_IMAGE)...$(NC)"
	@DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) docker build $(DOCKER_BUILD_ARGS) -t $(DOCKER_FULL_IMAGE) -t $(DOCKER_FULL_LATEST) .

.PHONY: docker-run
docker-run: docker-build ## Run Docker container
	$(ECHO) "$(BLUE)Running Docker container...$(NC)"
	@docker run -p 8085:8085 --rm $(DOCKER_FULL_LATEST)

.PHONY: docker-push
docker-push: docker-build ## Push Docker images
	$(ECHO) "$(BLUE)Pushing Docker images...$(NC)"
	@docker push $(DOCKER_FULL_IMAGE)
	@docker push $(DOCKER_FULL_LATEST)

.PHONY: ci
ci: deps fmt lint vet test build scan ## Run CI pipeline
	$(ECHO) "$(GREEN)CI pipeline completed successfully$(NC)"

.PHONY: scan
scan: ## Run security scan
	$(ECHO) "$(BLUE)Running security scan...$(NC)"
	@if ! command -v gosec &> /dev/null; then \
		echo "$(YELLOW)gosec not found, installing...$(NC)"; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest || { echo "$(RED)Failed to install gosec$(NC)"; exit 1; }; \
	fi
	@gosec -quiet ./...
	$(ECHO) "$(BLUE)Running vulnerability scan...$(NC)"
	@if ! command -v govulncheck &> /dev/null; then \
		echo "$(YELLOW)govulncheck not found, installing...$(NC)"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest || { echo "$(RED)Failed to install govulncheck$(NC)"; exit 1; }; \
	fi
	@govulncheck ./...

.PHONY: benchmark
benchmark: ## Run benchmarks
	$(ECHO) "$(BLUE)Running benchmarks...$(NC)"
	@go test -bench=. -benchmem ./...

.PHONY: tools-version
tools-version: ## Display tool versions
	@echo "$(BLUE)Go:$(NC) $(GO_VERSION)"
	@echo "$(BLUE)OS:$(NC) $(DETECTED_OS)"
	@echo "$(BLUE)Architecture:$(NC) $(GOARCH)"
	@echo "$(BLUE)golangci-lint:$(NC) $$(golangci-lint version 2>/dev/null || echo "not installed")"
	@echo "$(BLUE)gosec:$(NC) $$(gosec --version 2>/dev/null || echo "not installed")"
	@echo "$(BLUE)govulncheck:$(NC) $$(govulncheck --version 2>/dev/null || echo "not installed")"

.PHONY: tools-install
tools-install: ## Install development tools
	$(ECHO) "$(BLUE)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/boumenot/gocover-cobertura@latest
	$(ECHO) "$(GREEN)All tools installed successfully$(NC)"

.PHONY: generate
generate: ## Run code generation
	$(ECHO) "$(BLUE)Running code generation...$(NC)"
	@go generate ./...

.PHONY: mock
mock: ## Generate mocks
	$(ECHO) "$(BLUE)Generating mocks...$(NC)"
	@if ! command -v mockgen &> /dev/null; then \
		echo "$(YELLOW)mockgen not found, installing...$(NC)"; \
		go install github.com/golang/mock/mockgen@latest || { echo "$(RED)Failed to install mockgen$(NC)"; exit 1; }; \
	fi
	@go generate ./...

.PHONY: help
help: ## Show this help
	@echo "$(BLUE)Available commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'

# Documentation generation
.PHONY: docs
docs: ## Generate API documentation
	$(ECHO) "$(BLUE)Generating API documentation...$(NC)"
	@if ! command -v swag &> /dev/null; then \
		echo "$(YELLOW)swag not found, installing...$(NC)"; \
		go install github.com/swaggo/swag/cmd/swag@latest || { echo "$(RED)Failed to install swag$(NC)"; exit 1; }; \
	fi
	@swag init -g $(MAIN_PATH)/main.go -o ./docs/swagger

.DEFAULT_GOAL := help