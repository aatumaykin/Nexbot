# ==========================================
# Build & Deploy
# ==========================================

.PHONY: build build-all build-linux build-darwin build-darwin-arm build-windows install install-user release

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

release: build-all ## Create release packages with checksums
	@echo "📦 Creating release packages..."
	@cd $(OUTPUT_DIR) && for file in $(BINARY_NAME)-*; do \
		if [ -f "$$file" ]; then \
			shasum -a 256 "$$file" > "$$file.sha256" && \
			echo "✅ Created checksum: $$file.sha256"; \
		fi \
	done
	@echo "✅ Release packages ready in $(OUTPUT_DIR)/"
