# ==========================================
# Linting & Format
# ==========================================

.PHONY: lint vet fmt fmt-check ci

lint: ## Run golangci-lint
	golangci-lint run --timeout=5m

vet: ## Run go vet
	go vet ./...

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

fmt-check: ## Check code formatting
	@test -z "$$(gofmt -l .)" || (echo "❌ Code is not formatted. Run 'make fmt'" && exit 1)
	@echo "✅ Code is properly formatted"

ci: test-cover lint vet fmt-check ## Run all CI checks
