# ==========================================
# Project Variables
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

# Colors for output
NO_COLOR := \033[0m
OK_COLOR := \033[32;01m
ERROR_COLOR := \033[31;01m
WARN_COLOR := \033[33;01m
