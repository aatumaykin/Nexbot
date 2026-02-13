# ==========================================
# Makefile for Nexbot
# ==========================================

# Include all makefiles
include makefiles/includes.mk
include makefiles/development.mk
include makefiles/testing.mk
include makefiles/linting.mk
include makefiles/build.mk
include makefiles/cleanup.mk
include makefiles/utils.mk
include makefiles/docker.mk

# Default target
.DEFAULT_GOAL := help
