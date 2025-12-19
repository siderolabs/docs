# Makefile for Siderolabs documentation

# Variables
MINT_IMAGE := ghcr.io/siderolabs/docs-mint:latest
CONTAINER_NAME := docs-preview
PORT := 3000
DOCS_GEN_IMAGE := ghcr.io/siderolabs/docs-gen:latest
DOCS_CONVERT_IMAGE := ghcr.io/siderolabs/docs-convert:latest
TALOSCTL_IMAGE := ghcr.io/siderolabs/talosctl:v1.11.3
TALOS_VERSION := v1.11
VALE_IMAGE := jdkato/vale:latest
VALE_CONFIG ?= .vale.ini

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: build-mint
build-mint: ## Build the Mintlify documentation container locally
	docker build -t $(MINT_IMAGE) ./mintlify

.PHONY: docs-preview preview
docs-preview: ## Build and run the documentation preview server
	docker run --rm -it \
		--name $(CONTAINER_NAME) \
		-p $(PORT):$(PORT) \
		-v $(PWD)/public:/docs \
		$(MINT_IMAGE) dev

preview: docs-preview ## Alias for docs-preview

.PHONY: broken-links
broken-links: ## Run broken links check
	docker run --rm -t \
	  -v $(PWD)/public:/docs \
	  $(MINT_IMAGE) broken-links

docs.json: common.yaml omni.yaml ## Generate and validate docs.json from multiple config files
	docker pull $(DOCS_GEN_IMAGE)
	docker run --rm -v $(PWD):/workspace -w /workspace $(DOCS_GEN_IMAGE) \
		common.yaml \
		talos-v1.12.yaml \
		talos-v1.11.yaml \
		talos-v1.10.yaml \
		talos-v1.9.yaml \
		talos-v1.8.yaml \
		talos-v1.7.yaml \
		talos-v1.6.yaml \
		omni.yaml \
		kubernetes-guides.yaml \
		> public/docs.json

docs.json-local: common.yaml omni.yaml docs-gen/main.go ## Generate docs.json using local Go build
	cd docs-gen && go run . \
		../common.yaml \
		../talos-v1.12.yaml \
		../talos-v1.11.yaml \
		../talos-v1.10.yaml \
		../talos-v1.9.yaml \
		../talos-v1.8.yaml \
		../talos-v1.7.yaml \
		../talos-v1.6.yaml \
		../omni.yaml \
		../kubernetes-guides.yaml \
		> ../public/docs.json

.PHONY: check-missing
check-missing: ## Check for MDX files not included in config files
	docker run --rm -v $(PWD):/workspace -w /workspace $(DOCS_GEN_IMAGE) --detect-missing \
		common.yaml \
		talos-v1.12.yaml \
		talos-v1.11.yaml \
		talos-v1.10.yaml \
		talos-v1.9.yaml \
		talos-v1.8.yaml \
		talos-v1.7.yaml \
		talos-v1.6.yaml \
		omni.yaml \
		kubernetes-guides.yaml 

.PHONY: check-missing-local
check-missing-local: ## Check for missing files using local Go build
	cd docs-gen && go run . --detect-missing \
		../common.yaml \
		../talos-v1.12.yaml \
		../talos-v1.11.yaml \
		../talos-v1.10.yaml \
		../talos-v1.9.yaml \
		../talos-v1.8.yaml \
		../talos-v1.7.yaml \
		../talos-v1.6.yaml \
		../omni.yaml \
		../kubernetes-guides.yaml

.PHONY: generate-deps
generate-deps: ## Install Go dependencies for the generator
	cd docs-gen && go mod tidy

.PHONY: build-docs-gen-container
build-docs-gen-container: ## Build the docs-gen container locally
	docker build -t $(DOCS_GEN_IMAGE) ./docs-gen

.PHONY: build-docs-convert-container
build-docs-convert-container: ## Build the docs-convert container locally
	docker build -t $(DOCS_CONVERT_IMAGE) ./docs-convert

.PHONY: test-docs-gen
test-docs-gen: ## Run tests for the docs-gen utility
	cd docs-gen && go test -v

.PHONY: test-docs-gen-coverage
test-docs-gen-coverage: ## Run tests with coverage report
	cd docs-gen && go test -v -coverprofile=coverage.out \
		&& go tool cover -html=coverage.out -o coverage.html

.PHONY: test-docs-gen-race
test-docs-gen-race: ## Run tests with race detection
	cd docs-gen && go test -v -race

.PHONY: test-all
test-all: test-docs-gen ## Run all tests

.PHONY: generate-talos-reference
generate-talos-reference: ## Generate Talos reference docs and convert to MDX
	@echo "Generating Talos reference documentation..."
	docker pull $(TALOSCTL_IMAGE)
	docker pull $(DOCS_CONVERT_IMAGE)
	mkdir -p _out/docs
	docker run --rm -u $(shell id -u):$(shell id -g) -v $(PWD)/_out/docs:/docs $(TALOSCTL_IMAGE) docs /docs
	@echo "Converting generated docs to MDX..."
	docker run --rm -u $(shell id -u):$(shell id -g) -v $(PWD):/workspace $(DOCS_CONVERT_IMAGE) \
		/workspace/_out/docs /workspace/public/talos/$(TALOS_VERSION)/reference/configuration/
	rm -rf _out/docs
	@echo "Reference documentation generated in public/talos/$(TALOS_VERSION)/reference/configuration"

.PHONY: generate-talos-reference-local
generate-talos-reference-local: ## Generate Talos reference docs using local Go build
	@echo "Generating Talos reference documentation..."
	docker pull $(TALOSCTL_IMAGE)
	mkdir -p _out/docs
	docker run --rm -u $(shell id -u):$(shell id -g) -v $(PWD)/_out/docs:/docs $(TALOSCTL_IMAGE) docs /docs
	@echo "Converting generated docs to MDX..."
	cd docs-convert && go run main.go ../_out/docs ../public/talos/$(TALOS_VERSION)/reference/configuration/
	@echo "Reference documentation generated in public/talos/$(TALOS_VERSION)/reference/configuration/"

.PHONY: vale
vale: ## Run Vale on a file or directory: make vale DOC=public/path/to/file.mdx
	@if [ -z "$(DOC)" ]; then \
		echo "Usage: make vale DOC=public/path/or/file.mdx"; \
		exit 1; \
	fi
	@if [ ! -f "$(VALE_CONFIG)" ]; then \
		echo "$(VALE_CONFIG) not found at repo root."; \
		exit 1; \
	fi
	@echo "Running Vale on $(DOC)"
	docker run --rm -v $(PWD):/work -w /work $(VALE_IMAGE) \
		--config="$(VALE_CONFIG)" $(VALE_ARGS) "$(DOC)"

.PHONY: vale-changed
vale-changed: ## Run Vale on changed file vs HEAD
	@files="$$(git diff --name-only --diff-filter=AM HEAD | grep -E '\.mdx?$$|\.md$$' || true)"; \
	if [ -z "$$files" ]; then \
		echo "No changed Markdown/MDX files."; \
		exit 0; \
	fi; \
	echo "Linting changed files:" $$files; \
	docker run --rm -v $(PWD):/work -w /work $(VALE_IMAGE) \
		--config="$(VALE_CONFIG)" $(VALE_ARGS) $$files
