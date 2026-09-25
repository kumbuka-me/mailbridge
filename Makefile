.DEFAULT_GOAL := help

## Tool Versions

# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.14.0

# renovate: datasource=github-releases depName=gi8lino/dev-tools
DEV_TOOLS_VERSION ?= v0.9.0

# renovate: datasource=npm depName=prettier
PRETTIER_VERSION ?= 3.9.6


## Shared Development Tools

include bin/dev-tools.mk
include $(call dev-tools-module,tag)
include $(call dev-tools-module,help)


## Project Tools

GOLANGCI_LINT := bin/golangci-lint


## Formatting

NPX ?= npx
PRETTIER := $(NPX) --yes prettier@$(PRETTIER_VERSION)

PRETTIER_MD_SOURCES := README.md

PRETTIER_YAML_SOURCES := \
	.goreleaser.yaml \
	".github/**/*.{yml,yaml}"


## Build Configuration

BINARY ?= mailbridge
COMMAND ?= ./cmd/mailbridge

BUILD_VERSION ?= dev
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS ?= -s -w -X main.Version=$(BUILD_VERSION) -X main.Commit=$(BUILD_COMMIT)

RUN_ARGS ?=


## Tagging

# Compatibility alias for the shared current target.
.PHONY: tag
tag: current


##@ Development

.PHONY: download
download: dev-tools golangci-lint ## Download Go modules and development tools.
	go mod download

.PHONY: run
run: ## Run mailbridge locally.
	go run -ldflags="$(LDFLAGS)" $(COMMAND) $(RUN_ARGS)

.PHONY: build
build: ## Build the mailbridge binary.
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) $(COMMAND)

.PHONY: vet
vet: ## Run Go static analysis.
	go vet ./...

.PHONY: test
test: vet ## Run unit tests.
	go test -covermode=atomic -count=1 -parallel=4 -timeout=5m ./...

.PHONY: test-race
test-race: vet ## Run unit tests with the race detector.
	go test -race -count=1 -parallel=4 -timeout=5m ./...

.PHONY: cover
cover: ## Generate and display test coverage.
	go test \
		-coverprofile=coverage.out \
		-covermode=atomic \
		-count=1 \
		-parallel=4 \
		-timeout=5m \
		./...
	go tool cover -html=coverage.out

.PHONY: clean
clean: ## Clean generated files.
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf dist


##@ Formatting

.PHONY: fmt
fmt: fmt-go fmt-md fmt-yaml ## Format all supported files.

.PHONY: fmt-go
fmt-go: ## Format Go source files.
	go fmt ./...

.PHONY: fmt-md
fmt-md: ## Format Markdown files with Prettier.
	@$(PRETTIER) --write $(PRETTIER_MD_SOURCES)

.PHONY: fmt-yaml
fmt-yaml: ## Format YAML files with Prettier.
	@$(PRETTIER) --write $(PRETTIER_YAML_SOURCES)


##@ Linting

.PHONY: lint
lint: lint-go lint-md lint-yaml vet ## Run all linters and formatting checks.

.PHONY: lint-go
lint-go: golangci-lint ## Run golangci-lint.
	$(call run-tool,$(GOLANGCI_LINT),run)

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint and apply available fixes.
	$(call run-tool,$(GOLANGCI_LINT),run --fix)

.PHONY: lint-md
lint-md: ## Check Markdown formatting.
	@$(PRETTIER) --check $(PRETTIER_MD_SOURCES)

.PHONY: lint-yaml
lint-yaml: ## Check YAML formatting.
	@$(PRETTIER) --check $(PRETTIER_YAML_SOURCES)


##@ Dependencies

.PHONY: dev-tools
dev-tools: $(DEV_TAG) $(MAKE_HELP) $(GO_INSTALL_TOOL) ## Download the pinned shared development tools.

.PHONY: golangci-lint
golangci-lint: $(GO_INSTALL_TOOL) ## Download the pinned golangci-lint version.
	@$(GO_INSTALL_TOOL) \
		--target "$(GOLANGCI_LINT)" \
		--package github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
		--tool-version "$(GOLANGCI_LINT_VERSION)"
