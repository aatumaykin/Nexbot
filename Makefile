.PHONY: test test-cover lint build run clean build-all install version help

# ==========================================
# Build Variables
# ==========================================
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION ?= $(shell go version | awk '{print $$3}')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Binary output
BINARY_NAME ?= nexbot
MAIN_PATH ?= ./cmd/nexbot
OUTPUT_DIR ?= ./bin

# Cross-compilation targets
TARGET_OS ?= darwin linux windows
TARGET_ARCH ?= amd64 arm64

# Build flags
LDFLAGS ?= -s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.BuildTime=$(BUILD_TIME)' \
	-X 'main.GitCommit=$(GIT_COMMIT)' \
	-X 'main.GoVersion=$(GO_VERSION)'

# ==========================================
# Help
# ==========================================
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ==========================================
# Version
# ==========================================
version: ## Display version information
	@echo "Version:     $(VERSION)"
	@echo "Build Time:  $(BUILD_TIME)"
	@echo "Go Version:  $(GO_VERSION)"
	@echo "Git Commit:  $(GIT_COMMIT)"
	@echo "Binary:      $(BINARY_NAME)"

# ==========================================
# Build Targets
# ==========================================
build: ## Build binary for current platform
	@echo "🔨 Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(OUTPUT_DIR)
	@echo "Version: $(VERSION), Build Time: $(BUILD_TIME)"
	go build -v \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME) \
		$(MAIN_PATH)
	@echo "✅ Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"
	@ls -lh $(OUTPUT_DIR)/$(BINARY_NAME)

build-all: ## Build binary for all target platforms
	@echo "🔨 Building $(BINARY_NAME) for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	@$(foreach GOOS,$(TARGET_OS),\
		$(foreach GOARCH,$(TARGET_ARCH),\
			echo "Building $(GOOS)/$(GOARCH)..." && \
			GOOS=$(GOOS) GOARCH=$(GOARCH) go build -v \
				-ldflags "$(LDFLAGS)" \
				-o $(OUTPUT_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH) \
				$(MAIN_PATH) && \
			echo "✅ $(OUTPUT_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)" && \
			ls -lh $(OUTPUT_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH) && \
			echo ""; \
		)\
	)
	@echo "✅ All builds complete!"

build-linux: ## Build for Linux (amd64)
	@echo "🔨 Building for Linux amd64..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=amd64 go build -v \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 \
		$(MAIN_PATH)
	@echo "✅ Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64"

build-darwin: ## Build for macOS (amd64)
	@echo "🔨 Building for macOS amd64..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=amd64 go build -v \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 \
		$(MAIN_PATH)
	@echo "✅ Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64"

build-darwin-arm: ## Build for macOS (ARM64/Apple Silicon)
	@echo "🔨 Building for macOS ARM64..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=arm64 go build -v \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 \
		$(MAIN_PATH)
	@echo "✅ Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64"

build-windows: ## Build for Windows (amd64)
	@echo "🔨 Building for Windows amd64..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=amd64 go build -v \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe \
		$(MAIN_PATH)
	@echo "✅ Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# ==========================================
# Install Targets
# ==========================================
install: build ## Install binary to /usr/local/bin
	@echo "📦 Installing $(BINARY_NAME) to /usr/local/bin..."
	@$(MAKE) build
	@if [ -w /usr/local/bin ]; then \
		cp $(OUTPUT_DIR)/$(BINARY_NAME) /usr/local/bin/ && \
		chmod +x /usr/local/bin/$(BINARY_NAME) && \
		echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"; \
	else \
		echo "⚠️  Need sudo permissions. Please run: sudo make install"; \
		sudo cp $(OUTPUT_DIR)/$(BINARY_NAME) /usr/local/bin/ && \
		sudo chmod +x /usr/local/bin/$(BINARY_NAME) && \
		echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"; \
	fi
	@$(BINARY_NAME) --version || true

install-user: build ## Install binary to ~/bin
	@echo "📦 Installing $(BINARY_NAME) to $(HOME)/bin..."
	@$(MAKE) build
	@mkdir -p $(HOME)/bin
	@cp $(OUTPUT_DIR)/$(BINARY_NAME) $(HOME)/bin/ && \
		chmod +x $(HOME)/bin/$(BINARY_NAME) && \
		echo "✅ Installed to $(HOME)/bin/$(BINARY_NAME)"
	@if ! echo $$PATH | grep -q '$(HOME)/bin'; then \
		echo "⚠️  $(HOME)/bin is not in your PATH"; \
		echo "   Add this to your shell profile: export PATH=\"$$PATH:$(HOME)/bin\""; \
	fi
	@$(HOME)/bin/$(BINARY_NAME) --version || true

# ==========================================
# Release
# ==========================================
release: build-all ## Create release packages with checksums
	@echo "📦 Creating release packages..."
	@cd $(OUTPUT_DIR) && for file in $(BINARY_NAME)-*; do \
		if [ -f "$$file" ]; then \
			shasum -a 256 "$$file" > "$$file.sha256" && \
			echo "✅ Created checksum: $$file.sha256"; \
		fi \
	done
	@echo "✅ Release packages ready in $(OUTPUT_DIR)/"

# ==========================================
# Test targets
# ==========================================
test: ## Run all tests
	go test -v ./...

test-cover:
	go test -v -coverprofile=coverage.out -covermode=atomic ./...

# ==========================================
# Coverage reporting
# ==========================================
coverage-html:
	go test -coverprofile=coverage.out -covermode=atomic ./... && \
	go tool cover -html=coverage.out -o coverage.html

coverage-func:
	go test -coverprofile=coverage.out -covermode=atomic ./... && \
	go tool cover -func=coverage.out

coverage-check:
	@go test -coverprofile=coverage.out -covermode=atomic ./... && \
	TOTAL=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$TOTAL%"; \
	MIN_COVERAGE=60; \
	if [ $$(echo "$$TOTAL < $$MIN_COVERAGE" | bc) -eq 1 ]; then \
		echo "❌ Coverage ($$TOTAL%) is below threshold ($$MIN_COVERAGE%)"; \
		exit 1; \
	else \
		echo "✅ Coverage ($$TOTAL%) meets threshold ($$MIN_COVERAGE%)"; \
	fi

# ==========================================
# Linting
# ==========================================
lint:
	golangci-lint run --timeout=5m

vet:
	go vet ./...

# ==========================================
# Run targets
# ==========================================
run: ## Run server (nexbot serve)
	go run cmd/nexbot/main.go serve

# ==========================================
# Cleanup
# ==========================================
clean: ## Clean build artifacts
	@echo "🧹 Cleaning up..."
	@rm -rf $(OUTPUT_DIR)
	@find . -name "coverage.out" -delete
	@find . -name "coverage.html" -delete
	@echo "✅ Cleanup complete!"

clean-dist: clean ## Clean everything including dependencies
	@echo "🧹 Cleaning distribution..."
	@rm -rf vendor/
	@go clean -cache -testcache -modcache
	@echo "✅ Distribution cleanup complete!"

# ==========================================
# Dependencies
# ==========================================
deps: ## Download and verify dependencies
	go mod download
	go mod verify
	go mod tidy

deps-update: ## Update all dependencies
	go get -u ./...
	go mod tidy

# ==========================================
# Format
# ==========================================
fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

fmt-check: ## Check code formatting
	@test -z "$$(gofmt -l .)" || (echo "❌ Code is not formatted. Run 'make fmt'" && exit 1)
	@echo "✅ Code is properly formatted"

# ==========================================
# All checks
# ==========================================
ci: test-cover lint vet fmt-check ## Run all CI checks
