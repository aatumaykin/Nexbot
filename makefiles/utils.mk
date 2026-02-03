# ==========================================
# Utils
# ==========================================

.PHONY: deps deps-update version help

deps: ## Download and verify dependencies
	go mod download
	go mod verify
	go mod tidy

deps-update: ## Update all dependencies
	go get -u ./...
	go mod tidy

version: ## Display version information
	@echo "Version:     $(VERSION)"
	@echo "Build Time:  $(BUILD_TIME)"
	@echo "Go Version:  $(GO_VERSION)"
	@echo "Git Commit:  $(GIT_COMMIT)"
	@echo "Binary:      $(BINARY_NAME)"

help: ## Show this help message
	@echo '📖 Usage:'
	@echo '  make <target>'
	@echo ''
	@echo '🚀 Development:'
	@grep -hE '^(run|dev|serve|watch):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' || true
	@echo ''
	@echo '🧪 Testing:'
	@grep -hE '^(test|coverage).*?:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' || true
	@echo ''
	@echo '🔧 Linting & Format:'
	@grep -hE '^(lint|vet|fmt|ci|fmt-check):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' || true
	@echo ''
	@echo '📦 Build & Deploy:'
	@grep -hE '^(build|install|release).*?:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' || true
	@echo ''
	@echo '🧹 Cleanup:'
	@grep -hE '^clean.*?:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' || true
	@echo ''
	@echo '🔍 Utils:'
	@grep -hE '^(deps|version|help):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' || true
