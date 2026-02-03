# ==========================================
# Development
# ==========================================

.PHONY: run

run: ## Run server (nexbot serve)
	go run ./cmd/nexbot serve
