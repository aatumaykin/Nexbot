# ==========================================
# Cleanup
# ==========================================

.PHONY: clean clean-dist

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
