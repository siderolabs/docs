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
OMNI_CLI_GEN_IMAGE := ghcr.io/siderolabs/omni-cli-gen:latest
OMNI_CONFIG_GEN_IMAGE := ghcr.io/siderolabs/omni-config-gen:latest
MDX_NORMALIZE_IMAGE := ghcr.io/siderolabs/mdx-normalize:latest
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
OMNI_CLI_REF_PATH := public/omni/reference/cli.mdx
IMAGE_FACTORY_REF_PATH := public/omni/reference/image-factory-configuration.mdx
IMAGE_FACTORY_CONFIG_URL ?= https://raw.githubusercontent.com/siderolabs/image-factory/main/docs/configuration.md

# Frontmatter for the generated pages. Defined here (not read from the existing
# file) so a page is fully restored even if it was deleted or emptied.
OMNI_CLI_TITLE := omnictl CLI
OMNI_CLI_DESC := omnictl CLI tool reference.
IMAGE_FACTORY_TITLE := Image Factory Configuration
IMAGE_FACTORY_DESC := Complete reference for configuring Omni’s Image Factory service

# Pull an image only if it is not already present locally, so locally-built
# images (from the build-*-container targets) are usable before publishing.
pull_if_missing = docker image inspect $(1) >/dev/null 2>&1 || docker pull $(1)

# ---- Container image builds ------------------------------------------------

.PHONY: build-omni-cli-gen-container
build-omni-cli-gen-container: ## Build the omni-cli-gen container locally
	docker build -t $(OMNI_CLI_GEN_IMAGE) ./omni-cli-gen

.PHONY: build-omni-config-gen-container
build-omni-config-gen-container: ## Build the omni-config-gen container locally
	docker build -t $(OMNI_CONFIG_GEN_IMAGE) ./omni-config-gen

.PHONY: build-mdx-normalize-container
build-mdx-normalize-container: ## Build the mdx-normalize container locally
	docker build -t $(MDX_NORMALIZE_IMAGE) ./mdx-normalize

# ---- Normalization ---------------------------------------------------------

.PHONY: normalize-doc
normalize-doc: ## Normalize the generated Omni reference .mdx files for Mintlify (container)
	@$(call pull_if_missing,$(MDX_NORMALIZE_IMAGE))
	@if [ -f $(OMNI_CLI_REF_PATH) ]; then docker run --rm -i $(MDX_NORMALIZE_IMAGE) < $(OMNI_CLI_REF_PATH) > $(OMNI_CLI_REF_PATH).tmp && mv $(OMNI_CLI_REF_PATH).tmp $(OMNI_CLI_REF_PATH) || { rm -f $(OMNI_CLI_REF_PATH).tmp; exit 1; }; fi
	@if [ -f $(IMAGE_FACTORY_REF_PATH) ]; then docker run --rm -i $(MDX_NORMALIZE_IMAGE) --strip-hr < $(IMAGE_FACTORY_REF_PATH) > $(IMAGE_FACTORY_REF_PATH).tmp && mv $(IMAGE_FACTORY_REF_PATH).tmp $(IMAGE_FACTORY_REF_PATH) || { rm -f $(IMAGE_FACTORY_REF_PATH).tmp; exit 1; }; fi

.PHONY: normalize-doc-local
normalize-doc-local: ## Normalize the generated Omni reference .mdx files using local Go build
	@if [ -f $(OMNI_CLI_REF_PATH) ]; then cd mdx-normalize && go run . ../$(OMNI_CLI_REF_PATH); fi
	@if [ -f $(IMAGE_FACTORY_REF_PATH) ]; then cd mdx-normalize && go run . --strip-hr ../$(IMAGE_FACTORY_REF_PATH); fi

# ---- omnictl CLI reference -------------------------------------------------

.PHONY: generate-omni-cli-reference
generate-omni-cli-reference: ## Generate the omnictl CLI reference (container)
	@echo "Generating omnictl CLI reference..."
	@$(call pull_if_missing,$(OMNI_CLI_GEN_IMAGE))
	@tmp="$$(mktemp)"; \
	docker run --rm --entrypoint /bin/sh $(OMNI_CLI_GEN_IMAGE) \
		-c 'omnictl docs /tmp >/dev/null 2>&1 && cat /tmp/cli.md' > "$$tmp" \
		|| { echo "Error: 'omnictl docs' failed"; rm -f "$$tmp"; exit 1; }; \
	[ -s "$$tmp" ] || { echo "Error: omnictl did not produce cli.md"; rm -f "$$tmp"; exit 1; }; \
	{ \
		printf '%s\n' '---' 'title: $(OMNI_CLI_TITLE)' 'description: $(OMNI_CLI_DESC)' '---' ''; \
		awk '/^---[[:space:]]*$$/{c++; next} c>=2{print}' "$$tmp" \
			| sed '/^<!-- markdownlint-disable -->$$/d' \
			| awk 'NF{p=1} p'; \
	} > "$(OMNI_CLI_REF_PATH).tmp"; \
	mv "$(OMNI_CLI_REF_PATH).tmp" "$(OMNI_CLI_REF_PATH)"; \
	rm -f "$$tmp"
	@$(MAKE) --no-print-directory normalize-doc
	@echo "Reference documentation generated at $(OMNI_CLI_REF_PATH)"

.PHONY: generate-omni-cli-reference-local
generate-omni-cli-reference-local: ## Generate the omnictl CLI reference using local omnictl + Go build
	@echo "Generating omnictl CLI reference..."
	@command -v omnictl >/dev/null 2>&1 || { echo "Error: omnictl not found in PATH"; exit 1; }
	@tmp="$$(mktemp -d)"; \
	omnictl docs "$$tmp" >/dev/null || { echo "Error: 'omnictl docs' failed"; rm -rf "$$tmp"; exit 1; }; \
	[ -f "$$tmp/cli.md" ] || { echo "Error: omnictl did not produce cli.md"; rm -rf "$$tmp"; exit 1; }; \
	{ \
		printf '%s\n' '---' 'title: $(OMNI_CLI_TITLE)' 'description: $(OMNI_CLI_DESC)' '---' ''; \
		awk '/^---[[:space:]]*$$/{c++; next} c>=2{print}' "$$tmp/cli.md" \
			| sed '/^<!-- markdownlint-disable -->$$/d' \
			| awk 'NF{p=1} p'; \
	} > "$(OMNI_CLI_REF_PATH).tmp"; \
	mv "$(OMNI_CLI_REF_PATH).tmp" "$(OMNI_CLI_REF_PATH)"; \
	rm -rf "$$tmp"
	@$(MAKE) --no-print-directory normalize-doc-local
	@echo "Reference documentation generated at $(OMNI_CLI_REF_PATH)"

# ---- Omni configuration reference ------------------------------------------

.PHONY: generate-omni-config-reference
generate-omni-config-reference: ## Generate Omni configuration reference docs from JSON schema (container)
	@echo "Generating Omni configuration reference..."
	@$(call pull_if_missing,$(OMNI_CONFIG_GEN_IMAGE))
	docker run --rm $(OMNI_CONFIG_GEN_IMAGE) $(OMNI_CONFIG_SCHEMA_URL) > $(OMNI_CONFIG_REF_PATH).tmp && mv $(OMNI_CONFIG_REF_PATH).tmp $(OMNI_CONFIG_REF_PATH) || { rm -f $(OMNI_CONFIG_REF_PATH).tmp; exit 1; }
	@echo "Reference documentation generated at $(OMNI_CONFIG_REF_PATH)"

.PHONY: generate-omni-config-reference-local
generate-omni-config-reference-local: ## Generate Omni configuration reference docs using local Go build
	@echo "Generating Omni configuration reference..."
	cd omni-config-gen && go run . $(OMNI_CONFIG_SCHEMA_URL) > ../$(OMNI_CONFIG_REF_PATH).tmp && mv ../$(OMNI_CONFIG_REF_PATH).tmp ../$(OMNI_CONFIG_REF_PATH) || { rm -f ../$(OMNI_CONFIG_REF_PATH).tmp; exit 1; }
	@echo "Reference documentation generated at $(OMNI_CONFIG_REF_PATH)"

# ---- Image Factory configuration reference ---------------------------------

.PHONY: generate-omni-image-factory-reference
generate-omni-image-factory-reference: ## Generate the Image Factory configuration reference (container)
	@echo "Generating Image Factory configuration reference..."
	@tmp="$$(mktemp)"; \
	curl -fsSL "$(IMAGE_FACTORY_CONFIG_URL)" -o "$$tmp" || { echo "Error: failed to fetch $(IMAGE_FACTORY_CONFIG_URL)"; rm -f "$$tmp"; exit 1; }; \
	{ \
		printf '%s\n' '---' 'title: $(IMAGE_FACTORY_TITLE)' 'description: $(IMAGE_FACTORY_DESC)' '---' ''; \
		sed '1{/^# /d;}' "$$tmp" | awk 'NF{p=1} p'; \
	} > "$(IMAGE_FACTORY_REF_PATH).tmp"; \
	mv "$(IMAGE_FACTORY_REF_PATH).tmp" "$(IMAGE_FACTORY_REF_PATH)"; \
	rm -f "$$tmp"
	@$(MAKE) --no-print-directory normalize-doc
	@echo "Reference documentation generated at $(IMAGE_FACTORY_REF_PATH)"

.PHONY: generate-omni-image-factory-reference-local
generate-omni-image-factory-reference-local: ## Generate the Image Factory configuration reference using local Go build
	@echo "Generating Image Factory configuration reference..."
	@tmp="$$(mktemp)"; \
	curl -fsSL "$(IMAGE_FACTORY_CONFIG_URL)" -o "$$tmp" || { echo "Error: failed to fetch $(IMAGE_FACTORY_CONFIG_URL)"; rm -f "$$tmp"; exit 1; }; \
	{ \
		printf '%s\n' '---' 'title: $(IMAGE_FACTORY_TITLE)' 'description: $(IMAGE_FACTORY_DESC)' '---' ''; \
		sed '1{/^# /d;}' "$$tmp" | awk 'NF{p=1} p'; \
	} > "$(IMAGE_FACTORY_REF_PATH).tmp"; \
	mv "$(IMAGE_FACTORY_REF_PATH).tmp" "$(IMAGE_FACTORY_REF_PATH)"; \
	rm -f "$$tmp"
	@$(MAKE) --no-print-directory normalize-doc-local
	@echo "Reference documentation generated at $(IMAGE_FACTORY_REF_PATH)"

# ---- Aggregate -------------------------------------------------------------

.PHONY: generate-omni-reference
generate-omni-reference: generate-omni-cli-reference generate-omni-config-reference generate-omni-image-factory-reference ## Regenerate all Omni reference pages (containers)

.PHONY: generate-omni-reference-local
generate-omni-reference-local: generate-omni-cli-reference-local generate-omni-config-reference-local generate-omni-image-factory-reference-local ## Regenerate all Omni reference pages using local tools

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
