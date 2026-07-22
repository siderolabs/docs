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
STYLE_CHECK_IMAGE := ghcr.io/siderolabs/style-guide-checker:latest
TALOS_VERSION := v1.14
# Linter images are pinned to exact versions so `make code-review` (and CI) is
# reproducible: a new linter release can't turn the gate red on unchanged code.
# Bump these deliberately. Each is overridable from the environment (?=).
# checkmake has no semver tags, so it is pinned by image digest.
GOLANGCI_LINT_IMAGE ?= golangci/golangci-lint:v2.12.2
HADOLINT_IMAGE ?= hadolint/hadolint:v2.14.0
CHECKMAKE_IMAGE ?= cytopia/checkmake:latest@sha256:ff793674494e472661bc7b4cc623c9a172515ec011c6e802be586f101bd6c043

# Directories that never contain our own Go modules or Dockerfiles but are
# expensive to walk (public/ is large) or contain throwaway copies (worktrees).
# Pruning them keeps the discovery below fast on every make invocation.
DISCOVERY_PRUNE := \( -path ./.git -o -path ./public -o -path ./.claude -o -name node_modules -o -name vendor \) -prune

# Auto-discover every Go module in the repo (any directory with a go.mod) so
# `code-review` audits new programs automatically — just add a folder with a
# go.mod and it gets linted, no Makefile edit needed.
GO_MODULES := $(shell find . $(DISCOVERY_PRUNE) \
	-o -name go.mod -print | sed 's|^\./||; s|/go.mod$$||' | sort)

# Auto-discover every Dockerfile too, on the same principle: add one and it gets
# linted automatically.
DOCKERFILES := $(shell find . $(DISCOVERY_PRUNE) \
	-o -iname 'Dockerfile*' -print | sed 's|^\./||' | sort)

# Auto-fill the GitHub token from `gh` when it isn't already set in the environment.
# An exported/CI value wins; otherwise fall back to the local gh keychain. If gh is
# unavailable this is simply empty, matching the previous behaviour.
# `export` makes it visible to recipe subprocesses (e.g. the local `go run` targets),
# not just to the containers that pass it explicitly with `-e`.
GITHUB_TOKEN ?= $(shell gh auth token 2>/dev/null)
export GITHUB_TOKEN

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

# ---- Code review / linting -------------------------------------------------
#
# `code-review` audits ALL the tooling that builds the docs in one command,
# using the right linter for each language:
#   * Go code    -> golangci-lint (.golangci.yml). Catches "orphaned logic" and
#                   real bugs: `unused` (unused funcs/vars/types/fields),
#                   `unparam` (unused params), `ineffassign`/`wastedassign`
#                   (dead assignments), plus govet, staticcheck bug-checks,
#                   nilerr and bodyclose. Style-only linters are left off (see
#                   the .golangci.yml header for the rationale).
#   * Dockerfiles -> hadolint (.hadolint.yaml). Best-practice/correctness checks.
#   * Makefile    -> checkmake (.checkmake.ini). Structural best practices.
# Go modules and Dockerfiles are auto-discovered, so new programs are reviewed
# without editing this file. All three run even if one fails (so you see every
# problem at once), and the command exits non-zero on any finding — making it a
# drop-in CI gate.

.PHONY: code-review
code-review: ## Review all doc-building tooling: Go code, Dockerfiles, and the Makefile
	@$(call pull_if_missing,$(GOLANGCI_LINT_IMAGE))
	@$(call pull_if_missing,$(HADOLINT_IMAGE))
	@$(call pull_if_missing,$(CHECKMAKE_IMAGE))
	@failed=""; \
	for m in $(GO_MODULES); do \
		echo ""; echo "==> Go module: $$m"; \
		docker run --rm -v $(PWD):/workspace -w /workspace/$$m $(GOLANGCI_LINT_IMAGE) \
			golangci-lint run --config /workspace/.golangci.yml ./... || failed="$$failed go:$$m"; \
	done; \
	for f in $(DOCKERFILES); do \
		echo ""; echo "==> Dockerfile: $$f"; \
		docker run --rm -v $(PWD):/repo -w /repo $(HADOLINT_IMAGE) hadolint "$$f" || failed="$$failed dockerfile:$$f"; \
	done; \
	echo ""; echo "==> Makefile"; \
	docker run --rm -v $(PWD):/data -w /data $(CHECKMAKE_IMAGE) \
		--config=/data/.checkmake.ini Makefile || failed="$$failed makefile"; \
	echo ""; \
	if [ -n "$$failed" ]; then \
		echo "Code review FAILED. Issues in:$$failed"; \
		exit 1; \
	fi; \
	echo "Code review passed: all Go modules, Dockerfiles, and the Makefile are clean."

# talosctl is a multi-arch image and its `--arch` flag defaults to the running
# binary's architecture (runtime.GOARCH). Pin the platform so the generated
# reference docs are deterministic (matching CI) regardless of the contributor's
# machine — otherwise regenerating on Apple Silicon flips defaults to arm64.
TALOSCTL_PLATFORM := linux/amd64

.PHONY: generate-talos-reference
generate-talos-reference: ## Generate Talos reference docs and convert to MDX
	@echo "Generating Talos reference documentation..."
	docker pull --platform=$(TALOSCTL_PLATFORM) $(TALOSCTL_IMAGE)
	docker pull $(DOCS_CONVERT_IMAGE)
	mkdir -p _out/docs
	docker run --rm --platform=$(TALOSCTL_PLATFORM) -u $(shell id -u):$(shell id -g) -v $(PWD)/_out/docs:/docs $(TALOSCTL_IMAGE) docs /docs
	@echo "Converting generated docs to MDX..."
	docker run --rm -u $(shell id -u):$(shell id -g) -v $(PWD):/workspace $(DOCS_CONVERT_IMAGE) \
		/workspace/_out/docs /workspace/public/talos/$(TALOS_VERSION)/reference/configuration/
	rm -rf _out/docs
	@echo "Reference documentation generated in public/talos/$(TALOS_VERSION)/reference/configuration"

.PHONY: generate-talos-reference-local
generate-talos-reference-local: ## Generate Talos reference docs using local Go build
	@echo "Generating Talos reference documentation..."
	docker pull --platform=$(TALOSCTL_PLATFORM) $(TALOSCTL_IMAGE)
	mkdir -p _out/docs
	docker run --rm --platform=$(TALOSCTL_PLATFORM) -u $(shell id -u):$(shell id -g) -v $(PWD)/_out/docs:/docs $(TALOSCTL_IMAGE) docs /docs
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

# validate-tag distinguishes the two ways a TAG can be wrong, with a tailored
# message for each, and fails BEFORE the generator writes anything:
#   1. malformed  -> show the expected format and examples
#   2. unpublished -> the format is fine but no talosctl image exists for it yet
define validate-tag
	@echo "$(TAG)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)\.[0-9]+)?$$' || { \
		echo "Error: malformed TAG '$(TAG)'."; \
		echo "  Expected: vMAJOR.MINOR.PATCH with an optional -alpha.N / -beta.N / -rc.N suffix."; \
		echo "  Examples: v1.14.0   v1.14.0-alpha.0   v1.14.0-beta.2   v1.14.0-rc.1"; \
		exit 1; }
	@docker manifest inspect ghcr.io/siderolabs/talosctl:$(TAG) >/dev/null 2>&1 || { \
		echo "Error: TAG '$(TAG)' is well-formed but not published yet."; \
		echo "  No image 'ghcr.io/siderolabs/talosctl:$(TAG)' was found in the registry."; \
		echo "  The release may not be cut yet — see https://github.com/siderolabs/talos/releases"; \
		echo "  Pass a TAG whose talosctl image already exists."; \
		exit 1; }
endef

.PHONY: upgrade-talos-version
upgrade-talos-version: ## Upgrade Talos docs to a release tag: make upgrade-talos-version TAG=v1.14.0-beta.0
	@test -n "$(TAG)" || { echo "Error: TAG is required, e.g. make upgrade-talos-version TAG=v1.14.0-beta.0"; exit 1; }
	$(validate-tag)
	docker run --rm -v $(PWD):/workspace -w /workspace \
		-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
		$(VERSION_UPGRADE_IMAGE) --tag $(TAG)
	$(eval NEW_VERSION := $(shell cat .upgrade-version-tmp 2>/dev/null))
	@rm -f .upgrade-version-tmp
	$(MAKE) generate-talos-reference
	$(MAKE) changelog
	$(MAKE) docs.json
	$(MAKE) validate-docs-nav
	@echo ""
	@echo "Upgrade to $(TAG) complete! Run: make preview to preview your $(NEW_VERSION) docs"

.PHONY: upgrade-talos-version-local
upgrade-talos-version-local: ## Same as upgrade-talos-version but using the local Go build
	@test -n "$(TAG)" || { echo "Error: TAG is required, e.g. make upgrade-talos-version-local TAG=v1.14.0-beta.0"; exit 1; }
	$(validate-tag)
	cd version-upgrade-gen && go run . --workspace .. --tag $(TAG)
	$(eval NEW_VERSION := $(shell cat .upgrade-version-tmp 2>/dev/null))
	@rm -f .upgrade-version-tmp
	$(MAKE) generate-talos-reference-local
	$(MAKE) changelog
	$(MAKE) docs.json
	$(MAKE) validate-docs-nav
	@echo ""
	@echo "Upgrade to $(TAG) complete! Run: make preview to preview your $(NEW_VERSION) docs"

.PHONY: build-version-upgrade-container
build-version-upgrade-container: ## Build the version-upgrade-gen container locally
	docker build -t $(VERSION_UPGRADE_IMAGE) ./version-upgrade-gen

# ---- Style guide checker ---------------------------------------------------

# Extra flags passed to the checker, e.g. STYLE_CHECK_ARGS="-strict" or "-format github".
STYLE_CHECK_ARGS ?=

.PHONY: style-check
style-check: ## Check docs against the style guide (container). Scope with DOC=public/path
	@$(call pull_if_missing,$(STYLE_CHECK_IMAGE))
	docker run --rm -v $(PWD):/workspace -w /workspace $(STYLE_CHECK_IMAGE) $(STYLE_CHECK_ARGS) $(if $(DOC),$(DOC),public)

.PHONY: style-check-local
style-check-local: ## Check docs against the style guide using local Go build. Scope with DOC=public/path
	@cd style-guide-checker && go run . $(STYLE_CHECK_ARGS) ../$(if $(DOC),$(DOC),public)

# Git ref the "changed" target diffs against. Locally, HEAD catches your
# working-tree edits; in CI set this to the PR base, e.g. STYLE_CHECK_BASE=origin/main.
STYLE_CHECK_BASE ?= HEAD

.PHONY: style-check-changed
style-check-changed: ## Check changed .mdx files (container). Base: STYLE_CHECK_BASE (default HEAD)
	@$(call pull_if_missing,$(STYLE_CHECK_IMAGE))
	@files="$$( { git diff --name-only --diff-filter=AM $(STYLE_CHECK_BASE); git ls-files --others --exclude-standard; } | grep -E '\.mdx$$' | sort -u || true)"; \
	if [ -z "$$files" ]; then \
		echo "No changed MDX files."; \
		exit 0; \
	fi; \
	echo "Checking changed files:" $$files; \
	docker run --rm -v $(PWD):/workspace -w /workspace $(STYLE_CHECK_IMAGE) $(STYLE_CHECK_ARGS) $$files

.PHONY: style-check-changed-local
style-check-changed-local: ## Check changed .mdx files using local Go build. Base: STYLE_CHECK_BASE (default HEAD)
	@files="$$( { git diff --name-only --diff-filter=AM $(STYLE_CHECK_BASE); git ls-files --others --exclude-standard; } | grep -E '\.mdx$$' | sort -u || true)"; \
	if [ -z "$$files" ]; then \
		echo "No changed MDX files."; \
		exit 0; \
	fi; \
	echo "Checking changed files:" $$files; \
	cd style-guide-checker && go run . $(STYLE_CHECK_ARGS) $$(for f in $$files; do echo "../$$f"; done)

.PHONY: style-check-changed-auto
style-check-changed-auto: ## Check changed .mdx files, preferring local Go and falling back to the container.
	@files="$$( { git diff --name-only --diff-filter=AM $(STYLE_CHECK_BASE); git ls-files --others --exclude-standard; } | grep -E '\.mdx$$' | sort -u || true)"; \
	if [ -z "$$files" ]; then \
		echo "No changed MDX files."; \
		exit 0; \
	fi; \
	echo "Checking changed files:" $$files; \
	if command -v go >/dev/null 2>&1; then \
		cd style-guide-checker && go run . $(STYLE_CHECK_ARGS) $$(for f in $$files; do echo "../$$f"; done); \
	elif command -v docker >/dev/null 2>&1; then \
		echo "(go not found — using the container)"; \
		docker image inspect $(STYLE_CHECK_IMAGE) >/dev/null 2>&1 || docker build -q -t $(STYLE_CHECK_IMAGE) ./style-guide-checker >/dev/null; \
		docker run --rm -v $(PWD):/workspace -w /workspace $(STYLE_CHECK_IMAGE) $(STYLE_CHECK_ARGS) $$files; \
	else \
		echo "Skipping style check: neither go nor docker is available."; \
	fi

.PHONY: build-style-check-container
build-style-check-container: ## Build the style-guide-checker container locally
	docker build -t $(STYLE_CHECK_IMAGE) ./style-guide-checker
