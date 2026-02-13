# Docker commands for subagent

DOCKER_REGISTRY ?= 
IMAGE_NAME ?= nexbot/subagent
IMAGE_TAG ?= latest

.PHONY: docker-build-subagent
docker-build-subagent:
	@echo "Building subagent Docker image..."
	docker build -f Dockerfile.subagent -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "Image built: $(IMAGE_NAME):$(IMAGE_TAG)"

.PHONY: docker-build-subagent-multiarch
docker-build-subagent-multiarch:
	@echo "Building multi-architecture subagent image..."
	docker buildx build --platform linux/amd64,linux/arm64 \
		-f Dockerfile.subagent \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--push .
	@echo "Multi-arch image pushed: $(IMAGE_NAME):$(IMAGE_TAG)"

.PHONY: docker-push-subagent
docker-push-subagent:
	@echo "Pushing subagent image..."
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
	@echo "Image pushed: $(IMAGE_NAME):$(IMAGE_TAG)"

.PHONY: docker-tag-subagent
docker-tag-subagent:
	@if [ -z "$(new_tag)" ]; then \
		echo "Usage: make docker-tag-subagent NEW_TAG=v1.0.0"; \
		exit 1; \
	fi
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):$(new_tag)
	@echo "Tagged: $(IMAGE_NAME):$(new_tag)"

.PHONY: docker-test-subagent
docker-test-subagent:
	@echo "Testing subagent image..."
	@echo '{"version":"1.0","type":"ping","id":"test-1"}' | docker run --rm -i $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: docker-inspect-subagent
docker-inspect-subagent:
	@echo "Inspecting subagent image..."
	docker images $(IMAGE_NAME)
	docker inspect $(IMAGE_NAME):$(IMAGE_TAG) | jq '.[0].Size'
