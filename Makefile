# Build all by default, even if it's not first
.DEFAULT_GOAL := all

.PHONY: all
all: tidy format build

# ==============================================================================
# Build options

ROOT_PACKAGE=github.com/tenyee/simple-iam


# ==============================================================================
# Includes

include scripts/make-rules/golang.mk

# ==============================================================================
# Usage

define USAGE_OPTIONS

endef
export USAGE_OPTIONS

# ==============================================================================
# Targets

## build: Build source code for host platform
.PHONY: build
	@$(MAKE) go.build

.PHONY: tidy
tidy:
	@$(GO) mod tidy




## help: Show this help info.
.PHONY: help
help: Makefile
	@echo -e "\nUsage: make <TARGETS> <OPTIONS> ...\n\nTargets:"
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'
	@echo "$$USAGE_OPTIONS"