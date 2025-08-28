# Makefile for Siderolabs documentation

# Variables
MINT_IMAGE := ghcr.io/siderolabs/docs-mint:latest
CONTAINER_NAME := docs-preview
PORT := 3000
DOCS_GEN_IMAGE := ghcr.io/siderolabs/docs-gen:latest

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
		-v $(PWD):/docs \
		$(MINT_IMAGE) dev

preview: docs-preview ## Alias for docs-preview

.PHONY: broken-links
broken-links: ## Run broken links check
	docker run --rm -it \
		--name $(CONTAINER_NAME) \
		-p $(PORT):$(PORT) \
		-v $(PWD):/docs \
		$(MINT_IMAGE) broken-links

docs.json: common.yaml omni.yaml ## Generate and validate docs.json from multiple config files
	docker run --rm -v $(PWD):/workspace -w /workspace $(DOCS_GEN_IMAGE) \
		common.yaml \
		talos-v1.10.yaml \
		omni.yaml \
		> docs.json

docs.json-local: common.yaml omni.yaml docs-gen/main.go ## Generate docs.json using local Go build
	cd docs-gen && go run . \
		../common.yaml \
		../talos-v1.10.yaml \
		../omni.yaml \
		> ../docs.json

.PHONY: check-missing
check-missing: ## Check for MDX files not included in config files
	docker run --rm -v $(PWD):/workspace -w /workspace $(DOCS_GEN_IMAGE) --detect-missing \
		common.yaml \
		talos-v1.10.yaml \
		omni.yaml 

.PHONY: check-missing-local
check-missing-local: ## Check for missing files using local Go build
	cd docs-gen && go run . --detect-missing \
		../common.yaml \
		../talos-v1.10.yaml \
		../omni.yaml

.PHONY: generate-deps
generate-deps: ## Install Go dependencies for the generator
	cd docs-gen && go mod tidy

.PHONY: build-docs-gen-container
build-docs-gen-container: ## Build the docs-gen container locally
	docker build -t $(DOCS_GEN_IMAGE) ./docs-gen

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

