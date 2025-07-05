SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=
OS=$(shell uname -s)
LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always --dirty || echo dev) -X main.commit=$(shell git rev-parse HEAD || echo none)"

# Test data files
GO_SOURCES := $(shell find . -name '*.go' -not -path './vendor/*')
SHELL_TEMPLATES := internal/shell/*.sh internal/shell/*.tmpl.sh
TESTDATA_DIR := testdata
BINSTALLER_CONFIGS := $(wildcard $(TESTDATA_DIR)/*.binstaller.yml)
INSTALL_SCRIPTS := $(BINSTALLER_CONFIGS:.binstaller.yml=.install.sh)

# Schema files
SCHEMA_DIR := schema
TYPESPEC_SOURCES := $(SCHEMA_DIR)/main.tsp $(SCHEMA_DIR)/tspconfig.yaml
JSON_SCHEMA := $(SCHEMA_DIR)/output/@typespec/json-schema/InstallSpec.json
GENERATED_GO := pkg/spec/generated.go

export PATH := ./bin:$(PATH)
export GO111MODULE := on
# enable consistent Go 1.12/1.13 GOPROXY behavior.
export GOPROXY = https://proxy.golang.org

bin/goreleaser:
	mkdir -p bin
	GOBIN=$(shell pwd)/bin go install github.com/goreleaser/goreleaser/v2@latest

bin/golangci-lint:
	mkdir -p bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b ./bin v2.1.2

bin/shellcheck:
	mkdir -p bin
ifeq ($(OS), Darwin)
	curl -sfL -o ./bin/shellcheck https://github.com/caarlos0/shellcheck-docker/releases/download/v0.4.6/shellcheck_darwin
else
	curl -sfL -o ./bin/shellcheck https://github.com/caarlos0/shellcheck-docker/releases/download/v0.4.6/shellcheck
endif
	chmod +x ./bin/shellcheck

setup: bin/golangci-lint bin/shellcheck ## Install all the build and lint dependencies
	go mod download
.PHONY: setup

install: ## build and install
	go install $(LDFLAGS) ./cmd/binst

binst-init: binst ## generate .config/binstaller.yml from .config/goreleaser.yml
	./binst init --source goreleaser --file .config/goreleaser.yml --repo binary-install/binstaller

test: ## Run all the tests
	go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=2m

cover: test ## Run all the tests and opens the coverage report
	go tool cover -html=coverage.txt

fmt: ## gofmt and goimports all go files
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done

lint: bin/golangci-lint schema-lint ## Run all the linters
	./bin/golangci-lint run ./... --disable errcheck

ci: build test lint ## travis-ci entrypoint
	git diff .

build: ## Build a beta version of binstaller
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

test-run-installers-incremental: .testdata-timestamp $(INSTALL_SCRIPTS) ## Run only changed installer scripts
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

test-integration: test-gen-configs test-gen-installers test-run-installers ## Run full integration test suite
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

gen-schema: $(JSON_SCHEMA) ## Generate JSON Schema from TypeSpec definitions

gen-go: $(GENERATED_GO) ## Generate Go structs from JSON Schema

gen: gen-schema gen-go ## Generate both JSON Schema and Go structs

schema-lint: ## Format and lint TypeSpec schema files
	@echo "Formatting and linting schema files..."
	@cd $(SCHEMA_DIR) && npm run format && npm run deno:check

test-clean: ## Clean up test artifacts
	@echo "Cleaning test artifacts..."
	@rm -f $(TESTDATA_DIR)/*.install.sh .testdata-timestamp

.DEFAULT_GOAL := build

.PHONY: ci test help clean binst-init test-gen-configs test-gen-installers test-run-installers test-run-installers-incremental test-aqua-source test-all-platforms test-integration test-incremental test-clean gen-schema gen-go gen

clean: ## clean up everything
	go clean ./...
	rm -f binstaller binst
	rm -rf ./bin ./dist ./vendor
	git gc --aggressive

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
