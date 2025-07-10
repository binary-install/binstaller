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
JSON_SCHEMA := $(SCHEMA_DIR)/output/@typespec/json-schema/InstallSpec.json
YAML_SCHEMA := $(SCHEMA_DIR)/binstaller-schema.yaml
GENERATED_GO := pkg/spec/generated.go

# Aqua tool management - https://aquaproj.github.io/
AQUA_VERSION := v3.1.2
AQUA_INSTALLER_SHA256 := 9a5afb16da7191fbbc0c0240a67e79eecb0f765697ace74c70421377c99f0423
AQUA_ROOT_DIR ?= $(HOME)/.local/share/aquaproj-aqua

# Check if aqua is installed
AQUA_BIN := $(shell which aqua 2>/dev/null || echo "$(AQUA_ROOT_DIR)/bin/aqua")
export PATH := $(AQUA_ROOT_DIR)/bin:./bin:$(PATH)
export AQUA_DISABLE_LAZY_INSTALL := 1
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
	$(AQUA_BIN) install

setup: aqua-install ## Install all the build and lint dependencies
	go mod download
.PHONY: setup

install: ## build and install
	go install $(LDFLAGS) ./cmd/binst

binst-init: binst ## generate .config/binstaller.yml from .config/goreleaser.yml
	./binst init --source goreleaser --file .config/goreleaser.yml --repo binary-install/binstaller

test: test-unit ## Run unit tests (fast)

test-unit: ## Run unit tests without race detection or coverage
	go test $(TEST_OPTIONS) -failfast -short ./... -run $(TEST_PATTERN) -timeout=30s

test-race: ## Run unit tests with race detection
	go test $(TEST_OPTIONS) -failfast -short -race ./... -run $(TEST_PATTERN) -timeout=1m

test-cover: ## Run unit tests with coverage
	go test $(TEST_OPTIONS) -failfast -short -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt ./... -run $(TEST_PATTERN) -timeout=2m

test-all: ## Run all tests including E2E
	go test $(TEST_OPTIONS) -failfast -race ./... -run $(TEST_PATTERN) -timeout=5m

cover: test-cover ## Run all the tests with coverage and opens the coverage report
	go tool cover -html=coverage.txt

fmt: ## gofmt and goimports all go files
	find . -name '*.go' -type f | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done

lint: aqua-install schema-lint ## Run all the linters
	golangci-lint run ./... --disable errcheck

ci: build test-all lint fmt ## Run CI checks
	git diff .

build: ## Build binst binary
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
	@echo "Generating test configurations..."
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
	@echo "=== Testing with ignore patterns ==="
	@./binst check -c testdata/bat.binstaller.yml --ignore ".*-musl.*" --ignore "\.deb$$" || true
	@echo
	@echo "Check command tests completed"

test-integration: test-gen-configs test-gen-installers test-run-installers test-check ## Run full integration test suite
	@echo "Integration tests completed"

test-incremental: test-gen-installers test-run-installers-incremental ## Run incremental tests (only changed files)
	@echo "Incremental tests completed"

# Schema generation targets
$(JSON_SCHEMA): $(TYPESPEC_SOURCES)
	@echo "Generating JSON Schema from TypeSpec..."
	@cd $(SCHEMA_DIR) && npm install --silent && npm run gen:schema

$(GENERATED_GO): $(JSON_SCHEMA)
	@echo "Generating Go structs from JSON Schema..."
	@cd $(SCHEMA_DIR) && npm run gen:go

$(YAML_SCHEMA): $(JSON_SCHEMA) aqua-install
	@echo "Generating YAML Schema from JSON Schema..."
	@yq eval --input-format=json --output-format=yaml $(JSON_SCHEMA) > $(YAML_SCHEMA)

gen-schema: $(JSON_SCHEMA) ## Generate JSON Schema from TypeSpec definitions

gen-yaml-schema: $(YAML_SCHEMA) ## Generate YAML Schema from JSON Schema

gen-go: $(GENERATED_GO) ## Generate Go structs from JSON Schema

gen: gen-schema gen-yaml-schema gen-go gen-platforms ## Generate JSON Schema, YAML Schema, Go structs, and platform constants

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

.PHONY: ci test test-unit test-race test-cover test-all help clean binst-init test-gen-configs test-gen-installers test-run-installers test-run-installers-incremental test-aqua-source test-all-platforms test-integration test-incremental test-clean gen-schema gen-yaml-schema gen-go gen gen-platforms aqua-install

clean: ## clean up everything
	go clean ./...
	rm -f binstaller binst
	rm -rf ./bin ./dist
	git gc --aggressive

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
