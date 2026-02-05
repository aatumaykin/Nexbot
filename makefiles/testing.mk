# ==========================================
# Testing
# ==========================================

.PHONY: test test-cover coverage-html coverage-func coverage-check

test: ## Run all tests
	go test -v ./...

test-cover: ## Run tests with coverage
	@echo "Running tests with coverage (excluding tests package)..."
	go test -v -coverprofile=coverage.out -covermode=atomic ./cmd/... ./internal/...

coverage-html: ## Generate HTML coverage report
	go test -coverprofile=coverage.out -covermode=atomic ./... && \
	go tool cover -html=coverage.out -o coverage.html

coverage-func: ## Show coverage by function
	go test -coverprofile=coverage.out -covermode=atomic ./... && \
	go tool cover -func=coverage.out

coverage-check: ## Check if coverage meets threshold (60%)
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
