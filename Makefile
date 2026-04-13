SHELL := /bin/bash

.PHONY: fmt lint tool-%

export PATH := $(shell go tool bine path):$(PATH)

# List available module dependency updates.
deps: DEPS_FLAGS ?= -direct -update
deps: tool-go-mod-outdated
	go list -u -m -json all | go-mod-outdated $(DEPS_FLAGS)

# Run only the configured formatters (gofumpt, gci, etc.).
fmt: FMT_FLAGS ?=
fmt: tool-golangci-lint
	golangci-lint fmt $(FMT_FLAGS)

# Run linters and apply any auto-fixable issues, including formatter fixes.
lint: LINT_FLAGS ?= --fix=1
lint: tool-golangci-lint
	golangci-lint run $(LINT_FLAGS)

tool-%:
	@go tool bine get $* 1> /dev/null
