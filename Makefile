# Makefile for Siderolabs documentation

# Variables
MINT_IMAGE := ghcr.io/siderolabs/docs-mint:latest
CONTAINER_NAME := docs-preview
PORT := 3000
DOCS_GEN_IMAGE := ghcr.io/siderolabs/docs-gen:latest
DOCS_CONVERT_IMAGE := ghcr.io/siderolabs/docs-convert:latest
CHANGELOG_GEN_IMAGE := ghcr.io/siderolabs/changelog-gen:latest
VERSION_UPGRADE_IMAGE := ghcr.io/siderolabs/version-upgrade-gen:latest
TALOSCTL_IMAGE := ghcr.io/siderolabs/talosctl:v1.14.0-alpha.2
TALOS_VERSION := v1.14
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
		talos-v1.13.yaml \
		talos-v1.12.yaml \
		talos-v1.11.yaml \
		talos-v1.10.yaml \
		talos-v1.9.yaml \
		talos-v1.8.yaml \
		talos-v1.7.yaml \
		omni.yaml \
		kubernetes-guides.yaml \
		changelog.yaml \
		> public/docs.json

docs.json-local: common.yaml omni.yaml docs-gen/main.go ## Generate docs.json using local Go build
	cd docs-gen && go run . \
		../common.yaml \
		../talos-v1.13.yaml \
		../talos-v1.12.yaml \
		../talos-v1.11.yaml \
		../talos-v1.10.yaml \
		../talos-v1.9.yaml \
		../talos-v1.8.yaml \
		../talos-v1.7.yaml \
		../omni.yaml \
		../kubernetes-guides.yaml \
		../changelog.yaml \
		> ../public/docs.json

.PHONY: check-missing
check-missing: ## Check for MDX files not included in config files
	docker run --rm -v $(PWD):/workspace -w /workspace $(DOCS_GEN_IMAGE) --detect-missing \
		common.yaml \
		talos-v1.13.yaml \
		talos-v1.12.yaml \
		talos-v1.11.yaml \
		talos-v1.10.yaml \
		talos-v1.9.yaml \
		talos-v1.8.yaml \
		talos-v1.7.yaml \
		omni.yaml \
		kubernetes-guides.yaml \
		changelog.yaml

.PHONY: check-missing-local
check-missing-local: ## Check for missing files using local Go build
	cd docs-gen && go run . --detect-missing \
		../common.yaml \
		../talos-v1.13.yaml \
		../talos-v1.12.yaml \
		../talos-v1.11.yaml \
		../talos-v1.10.yaml \
		../talos-v1.9.yaml \
		../talos-v1.8.yaml \
		../talos-v1.7.yaml \
		../omni.yaml \
		../kubernetes-guides.yaml \
		../changelog.yaml

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

OMNI_CONFIG_SCHEMA_URL ?= https://raw.githubusercontent.com/siderolabs/omni/refs/heads/main/internal/pkg/config/schema.json
OMNI_CONFIG_REF_PATH := public/omni/reference/omni-configuration.mdx

.PHONY: generate-omni-config-reference
generate-omni-config-reference: ## Generate Omni configuration reference docs from JSON schema
	@echo "Generating Omni configuration reference..."
	cd omni-config-gen && go run . $(OMNI_CONFIG_SCHEMA_URL) > ../$(OMNI_CONFIG_REF_PATH)
	@echo "Reference documentation generated at $(OMNI_CONFIG_REF_PATH)"

OMNI_CLI_REF_PATH := public/omni/reference/cli.mdx
IMAGE_FACTORY_REF_PATH := public/omni/reference/image-factory-configuration.mdx
IMAGE_FACTORY_CONFIG_URL ?= https://raw.githubusercontent.com/siderolabs/image-factory/main/docs/configuration.md

.PHONY: normalize-doc
normalize-doc: ## Normalize the generated Omni reference .mdx files for Mintlify (fence indented code, strip --- rules)
	cd mdx-normalize && go run . ../$(OMNI_CLI_REF_PATH)
	cd mdx-normalize && go run . --strip-hr ../$(IMAGE_FACTORY_REF_PATH)

.PHONY: generate-omni-cli-reference
generate-omni-cli-reference: ## Generate the omnictl CLI reference (public/omni/reference/cli.mdx)
	@echo "Generating omnictl CLI reference..."
	@command -v omnictl >/dev/null 2>&1 || { echo "Error: omnictl not found in PATH"; exit 1; }
	@tmp="$$(mktemp -d)"; \
	omnictl docs "$$tmp" >/dev/null || { echo "Error: 'omnictl docs' failed"; rm -rf "$$tmp"; exit 1; }; \
	[ -f "$$tmp/cli.md" ] || { echo "Error: omnictl did not produce cli.md"; rm -rf "$$tmp"; exit 1; }; \
	{ \
		awk '/^---[[:space:]]*$$/{c++; print; if(c==2) exit; next} c>=1{print}' "$(OMNI_CLI_REF_PATH)"; \
		echo ""; \
		awk '/^---[[:space:]]*$$/{c++; next} c>=2{print}' "$$tmp/cli.md" \
			| sed '/^<!-- markdownlint-disable -->$$/d' \
			| awk 'NF{p=1} p'; \
	} > "$(OMNI_CLI_REF_PATH).tmp"; \
	mv "$(OMNI_CLI_REF_PATH).tmp" "$(OMNI_CLI_REF_PATH)"; \
	rm -rf "$$tmp"
	@$(MAKE) --no-print-directory normalize-doc
	@echo "Reference documentation generated at $(OMNI_CLI_REF_PATH)"

.PHONY: generate-omni-image-factory-reference
generate-omni-image-factory-reference: ## Generate the Image Factory configuration reference (public/omni/reference/image-factory-configuration.mdx)
	@echo "Generating Image Factory configuration reference..."
	@tmp="$$(mktemp)"; \
	curl -fsSL "$(IMAGE_FACTORY_CONFIG_URL)" -o "$$tmp" || { echo "Error: failed to fetch $(IMAGE_FACTORY_CONFIG_URL)"; rm -f "$$tmp"; exit 1; }; \
	{ \
		awk '/^---[[:space:]]*$$/{c++; print; if(c==2) exit; next} c>=1{print}' "$(IMAGE_FACTORY_REF_PATH)"; \
		echo ""; \
		sed '1{/^# /d;}' "$$tmp" | awk 'NF{p=1} p'; \
	} > "$(IMAGE_FACTORY_REF_PATH).tmp"; \
	mv "$(IMAGE_FACTORY_REF_PATH).tmp" "$(IMAGE_FACTORY_REF_PATH)"; \
	rm -f "$$tmp"
	@$(MAKE) --no-print-directory normalize-doc
	@echo "Reference documentation generated at $(IMAGE_FACTORY_REF_PATH)"

.PHONY: generate-omni-reference
generate-omni-reference: generate-omni-cli-reference generate-omni-config-reference generate-omni-image-factory-reference ## Regenerate all Omni reference pages (omnictl CLI + configuration + Image Factory)

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

.PHONY: changelog
changelog: ## Generate the changelog from GitHub releases
	docker pull $(CHANGELOG_GEN_IMAGE)
	docker run --rm -v $(PWD):/workspace -w /workspace \
		-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
		$(CHANGELOG_GEN_IMAGE) --output public/changelog.mdx

.PHONY: changelog-local
changelog-local: ## Generate the changelog using local Go build
	cd changelog-gen && go run . --output ../public/changelog.mdx

.PHONY: validate-docs-nav
validate-docs-nav: ## Validate all talos yaml nav configs match their content directories
	cd docs-validate && go run . --workspace ..

.PHONY: upgrade-talos-version
upgrade-talos-version: ## Upgrade docs to the next Talos minor version (fetches versions from GitHub)
	docker run --rm -it -v $(PWD):/workspace -w /workspace \
		-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
		$(VERSION_UPGRADE_IMAGE)
	$(eval NEW_VERSION := $(shell cat .upgrade-version-tmp))
	@rm -f .upgrade-version-tmp
	$(MAKE) changelog
	$(MAKE) docs.json
	$(MAKE) validate-docs-nav
	@echo ""
	@echo "Upgrade to $(NEW_VERSION) complete! Run: make preview to preview your $(NEW_VERSION) docs"

.PHONY: upgrade-talos-version-local
upgrade-talos-version-local: ## Upgrade docs to the next Talos minor version using local Go build
	cd version-upgrade-gen && go run . --workspace ..
	$(eval NEW_VERSION := $(shell cat .upgrade-version-tmp))
	@rm -f .upgrade-version-tmp
	$(MAKE) changelog
	$(MAKE) docs.json
	$(MAKE) validate-docs-nav
	@echo ""
	@echo "Upgrade to $(NEW_VERSION) complete! Run: make preview to preview your $(NEW_VERSION) docs"

.PHONY: build-version-upgrade-container
build-version-upgrade-container: ## Build the version-upgrade-gen container locally
	docker build -t $(VERSION_UPGRADE_IMAGE) ./version-upgrade-gen
