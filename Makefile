# Makefile for binstaller
#
# Key targets:
#   make ci              - Run all CI checks without external API calls (safe for offline use)
#   make test-integration - Run full integration tests (accesses GitHub API, generates test configs/installers)
#
# Note: For test-integration, optionally set GITHUB_TOKEN environment variable to avoid GitHub API rate limits:
#   GITHUB_TOKEN=your_token make test-integration

SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=
OS=$(shell uname -s)
LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always --dirty || echo dev) -X main.commit=$(shell git rev-parse HEAD || echo none)"

# Test data files
GO_SOURCES := $(shell find . -name '*.go' -type f)
SHELL_TEMPLATES := internal/shell/*.sh internal/shell/*.tmpl.sh
TESTDATA_DIR := testdata
BINSTALLER_CONFIGS := $(wildcard $(TESTDATA_DIR)/*.binstaller.yml)
INSTALL_SCRIPTS := $(BINSTALLER_CONFIGS:.binstaller.yml=.install.sh)

# Schema files
SCHEMA_DIR := schema
TYPESPEC_SOURCES := $(SCHEMA_DIR)/main.tsp $(SCHEMA_DIR)/tspconfig.yaml
TYPESPEC_OUTPUT := $(SCHEMA_DIR)/output/@typespec/json-schema/InstallSpec.json
JSON_SCHEMA := $(SCHEMA_DIR)/InstallSpec.json
YAML_SCHEMA := $(SCHEMA_DIR)/InstallSpec.yaml
GENERATED_GO := pkg/spec/generated.go

# Aqua tool management - https://aquaproj.github.io/
AQUA_VERSION := v3.1.2
AQUA_INSTALLER_SHA256 := 9a5afb16da7191fbbc0c0240a67e79eecb0f765697ace74c70421377c99f0423
AQUA_ROOT_DIR ?= $(HOME)/.local/share/aquaproj-aqua

# Check if aqua is installed
AQUA_BIN := $(shell which aqua 2>/dev/null || echo "$(AQUA_ROOT_DIR)/bin/aqua")
export PATH := $(AQUA_ROOT_DIR)/bin:./bin:$(PATH)
export GO111MODULE := on
# Use Go module proxy
export GOPROXY = https://proxy.golang.org

# Install aqua if not present
$(AQUA_BIN):
	@echo "Installing aqua $(AQUA_VERSION)..."
	@curl -sSfL -O https://raw.githubusercontent.com/aquaproj/aqua-installer/$(AQUA_VERSION)/aqua-installer
	@echo "$(AQUA_INSTALLER_SHA256)  aqua-installer" | sha256sum -c -
	@chmod +x aqua-installer
	@./aqua-installer
	@rm -f aqua-installer
	@echo "aqua installed successfully"
	@# Add aqua to GITHUB_PATH if running in GitHub Actions
	@if [ -n "$$GITHUB_PATH" ]; then \
		echo "$(AQUA_ROOT_DIR)/bin" >> $$GITHUB_PATH; \
		echo "Added aqua to GITHUB_PATH"; \
	fi

aqua-install: $(AQUA_BIN) ## Install tools via aqua
	$(AQUA_BIN) install --only-link

setup: aqua-install ## Install all the build and lint dependencies
	go mod download
.PHONY: setup

install: ## build and install
	go install $(LDFLAGS) ./cmd/binst

binst-init: binst ## generate .config/binstaller.yml from .config/goreleaser.yml
	./binst init --source goreleaser --file .config/goreleaser.yml --repo binary-install/binstaller

test: test-unit ## Run unit tests

test-unit: ## Run unit tests without race detection or coverage
	go test $(TEST_OPTIONS) -failfast ./... -run $(TEST_PATTERN) -timeout=30s

test-race: ## Run unit tests with race detection
	go test $(TEST_OPTIONS) -failfast -race ./... -run $(TEST_PATTERN) -timeout=1m

test-cover: ## Run unit tests with coverage
	go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt ./... -run $(TEST_PATTERN) -timeout=2m

cover: test-cover ## Run all the tests with coverage and opens the coverage report
	go tool cover -html=coverage.txt

fmt:
	find . -name '*.go' -type f | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done
	git ls-files -c -o --exclude-standard | xargs nllint -trim-space -trim-trailing-space -fix -ignore-notfound

lint: aqua-install schema-lint ## Run all the linters
	golangci-lint run ./... --disable errcheck

ci: build test lint gen fmt ## Run CI checks (no external API calls)
	@echo "Checking for uncommitted changes..."
	@if [ -n "$$(git status -s)" ]; then \
		echo "Warning: Uncommitted changes detected (possibly generated files):"; \
		git status -s; \
		echo "Please check and commit if necessary."; \
	else \
		echo "No uncommitted changes - all good!"; \
	fi

build: setup ## Build binst binary
	go build $(LDFLAGS) ./cmd/binst

# Binary with dependency tracking (includes embedded shell templates and generated types)
binst: $(GO_SOURCES) $(SHELL_TEMPLATES) $(GENERATED_GO) go.mod go.sum
	@echo "Building binst binary..."
	go build $(LDFLAGS) -o binst ./cmd/binst

# Install script generation with incremental builds
$(TESTDATA_DIR)/%.install.sh: $(TESTDATA_DIR)/%.binstaller.yml binst
	@echo "Generating installer for $*..."
	./binst gen --config $< -o $@

# Test targets
test-gen-configs: binst ## Generate test configuration files
	@./test/gen_config.sh

test-gen-installers: $(INSTALL_SCRIPTS) ## Generate installer scripts (incremental)
	@echo "Generated installer scripts"

# Test execution with timestamp tracking
.testdata-timestamp:
	@touch .testdata-timestamp

test-run-installers: ## Run all installer scripts in parallel
	@echo "Running installer scripts..."
	@./test/run_installers.sh
	@touch .testdata-timestamp

test-run-installers-incremental: $(AQUA_BIN) aqua-install .testdata-timestamp $(INSTALL_SCRIPTS) ## Run only changed installer scripts
	@echo "Running incremental installer tests..."
	@CHANGED_SCRIPTS=$$(find $(TESTDATA_DIR) -name "*.install.sh" -newer .testdata-timestamp 2>/dev/null || echo ""); \
	if [ -n "$$CHANGED_SCRIPTS" ]; then \
		echo "Testing changed installers: $$CHANGED_SCRIPTS"; \
		TMPDIR=$$(mktemp -d); \
		trap 'rm -rf -- "$$TMPDIR"' EXIT HUP INT TERM; \
		echo "$$CHANGED_SCRIPTS" | tr ' ' '\n' | rush -j5 -k "{} -b $$TMPDIR"; \
	else \
		echo "No installer scripts have changed since last test"; \
	fi
	@touch .testdata-timestamp

test-aqua-source: binst ## Test aqua registry source integration
	@echo "Testing aqua source..."
	@./test/aqua_source.sh

test-all-platforms: binst ## Test reviewdog installer across all supported platforms
	@echo "Testing all supported platforms..."
	@./test/all-supported-platforms-reviewdog.sh

test-check: binst ## Test check command with various configurations
	@echo "Testing check command..."
	@echo "=== Testing default config ==="
	@./binst check
	@echo
	@echo "=== Testing with specific version ==="
	@./binst check --version v0.2.0
	@echo
	@echo "=== Testing without asset verification ==="
	@./binst check --check-assets=false
	@echo
	@echo "=== Testing with ignore patterns (external repo) ==="
	@./binst check -c testdata/bat.binstaller.yml --version v0.24.0 \
		--ignore ".*-musl.*" \
		--ignore "\.deb$$" \
		--ignore ".*-arm-.*" \
		--ignore ".*-i686-.*" \
		--ignore ".*-windows-.*"
	@echo
	@echo "Check command tests completed"

test-target-version: binst ## Test target version functionality
	@echo "Testing target version functionality..."
	@./test/target_version.sh

test-runner-mode: binst ## Test runner mode functionality
	@echo "Testing runner mode functionality..."
	@./test/runner_mode.sh

test-template-validation: binst ## Test template validation security feature
	@echo "Testing template validation..."
	@./test/template_validation_test.sh

test-comprehensive-validation: binst ## Test comprehensive field validation
	@echo "Testing comprehensive field validation..."
	@./test/comprehensive_validation_test.sh

test-integration: test-gen-configs test-gen-installers test-run-installers test-check test-target-version test-runner-mode test-template-validation test-comprehensive-validation fmt ## Run full integration test suite (accesses GitHub API)
	@echo "Integration tests completed"
	@echo "Note: These tests access GitHub API. Set GITHUB_TOKEN to avoid rate limits."

test-incremental: test-gen-installers test-run-installers-incremental ## Run incremental tests (only changed files)
	@echo "Incremental tests completed"

# Schema generation targets
$(TYPESPEC_OUTPUT): $(TYPESPEC_SOURCES)
	@echo "Generating JSON Schema from TypeSpec..."
	@cd $(SCHEMA_DIR) && npm install --silent && npm run gen:schema

$(JSON_SCHEMA): $(TYPESPEC_OUTPUT)
	@echo "Copying JSON Schema to schema root..."
	@cp $(TYPESPEC_OUTPUT) $(JSON_SCHEMA)

$(GENERATED_GO): $(JSON_SCHEMA)
	@echo "Generating Go structs from JSON Schema..."
	@cd $(SCHEMA_DIR) && npm install --silent && npm run gen:go

$(YAML_SCHEMA): $(JSON_SCHEMA) aqua-install
	@echo "Generating YAML Schema from JSON Schema..."
	@yq eval --input-format=json --output-format=yaml $(JSON_SCHEMA) > $(YAML_SCHEMA)

gen-schema: $(JSON_SCHEMA) ## Generate JSON Schema from TypeSpec definitions

gen-yaml-schema: $(YAML_SCHEMA) ## Generate YAML Schema from JSON Schema

gen-go: $(GENERATED_GO) ## Generate Go structs from JSON Schema

gen: gen-schema gen-yaml-schema gen-go gen-platforms fmt ## Generate JSON Schema, YAML Schema, Go structs, and platform constants

gen-platforms: ## Generate platform constants from TypeSpec
	@echo "Generating platform constants from TypeSpec..."
	@cd $(SCHEMA_DIR) && npm run gen:platforms
	@echo "Formatting generated Go code..."
	@go fmt pkg/asset/platforms_generated.go

schema-lint: ## Format and lint TypeSpec schema files
	@echo "Installing schema dependencies..."
	@cd $(SCHEMA_DIR) && npm install --silent
	@echo "Formatting and linting schema files..."
	@cd $(SCHEMA_DIR) && npm run format && npm run deno:check

test-clean: ## Clean up test artifacts
	@echo "Cleaning test artifacts..."
	@rm -f $(TESTDATA_DIR)/*.install.sh .testdata-timestamp

.DEFAULT_GOAL := build

.PHONY: ci test test-unit test-race test-cover help clean binst-init test-gen-configs test-gen-installers test-run-installers test-run-installers-incremental test-aqua-source test-all-platforms test-integration test-incremental test-clean test-target-version test-runner-mode test-template-validation test-comprehensive-validation gen-schema gen-yaml-schema gen-go gen gen-platforms aqua-install

clean: ## clean up everything
	go clean ./...
	rm -f binstaller binst
	rm -rf ./bin ./dist
	rm -f $(JSON_SCHEMA) $(YAML_SCHEMA)
	git gc --aggressive

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
