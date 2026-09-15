# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains the documentation for Siderolabs products, particularly Omni - a Kubernetes management platform and Talos - a Linux operating system for Kubernetes. The documentation is built using Mintlify and configured via `docs.json`.

## Architecture

### Documentation Structure
- **Single documentation site**: All content lives under `public/`, in the `omni/`, `talos/`, and `kubernetes-guides/` directories.
- **Mintlify-based**: Uses Mintlify documentation platform with configuration in `public/docs.json`
- **MDX format**: All documentation files use `.mdx` extension for enhanced markdown with React components
- **Hierarchical organization**: Content is organized into logical groups (Overview, Getting Started, Infrastructure, etc.)

### Key Directories
- `public/omni/overview/` - High-level product information
- `public/omni/getting-started/` - User onboarding guides
- `public/omni/infrastructure-and-extensions/` - Infrastructure setup and extensions
- `public/omni/omni-cluster-setup/` - Cluster configuration guides
- `public/omni/cluster-management/` - Ongoing cluster operations
- `public/omni/security-and-authentication/` - Security and auth configuration
- `public/omni/reference/` - Reference documentation, much of it generated
- `public/images/` - Static assets and screenshots

### Navigation Configuration

**`public/docs.json` is generated. Never hand-edit it.** Navigation is defined in the per-product YAML files at the repository root: `omni.yaml`, `talos-v*.yaml`, `kubernetes-guides.yaml`, `common.yaml`, and `changelog.yaml`. Every page must be listed in one of them to appear on the site.

`make docs.json` compiles those YAML files into `public/docs.json`, which **is** committed. CI regenerates it and fails if the result differs from the committed copy, so **run `make docs.json` and commit the result in the same change whenever you touch a nav YAML or add a page.**

Note that `make check-missing` passing is not enough on its own. It only checks that every `.mdx` file is referenced by some YAML, and says nothing about whether the committed `docs.json` is current.

## Content Standards

### File Naming
- Use kebab-case for file and directory names
- All documentation files use `.mdx` extension
- Image files are organized in `images/` subdirectories within relevant sections

### Documentation Structure
- Each `.mdx` file begins with YAML frontmatter containing at minimum a `title` field
- Content focuses on Omni platform usage, Kubernetes cluster management, and Talos Linux integration
- Includes step-by-step guides with screenshots stored in adjacent `images/` directories

## Development Workflow

### Before you push

CI (`.github/workflows/docs-ci.yaml`) runs exactly four checks. Run the same four locally and a red PR becomes rare:

```bash
make broken-links                                    # every internal link resolves
make validate-docs-nav                               # nav YAMLs match the content directories
make style-check-changed STYLE_CHECK_BASE=upstream/main   # style guide, changed files only
make docs.json && git diff --exit-code               # the committed docs.json is current
```

Only the style check reports at two levels. Warnings do not fail CI, errors do, and plenty of existing pages carry warnings, so check whether a warning is on a line you actually touched before chasing it.

Most targets run in a container pulled from ghcr.io. When an image is unavailable, the `-local` variants (`style-check-local`, `docs.json-local`, `check-missing-local`) build the Go tool instead and are equivalent. `make help` lists everything.

### Local Development with Docker
To preview the documentation locally without installing Mintlify, use the provided Makefile:

```bash
# Build container and start preview server
make preview

# View available commands
make help
```

Alternatively, use Docker commands directly:
```bash
# Build the Docker image
docker build -t mintlify-docs .

# Run the development server (mounts current directory)
docker run -p 3000:3000 -v $(pwd):/docs mintlify-docs
```

Access the site at http://localhost:3000

### Generating docs.json
The `docs.json` file is automatically generated from multiple YAML config files using a containerized generator:

```bash
# Generate and validate docs.json using container (recommended)
make docs.json

# Check for MDX files not included in configuration
make check-missing

# Build the docs-gen container locally (if needed)
make build-docs-gen

# Alternative: Use local Go build for development
make docs.json-local
make check-missing-local
make generate-deps  # Install Go dependencies
```

The generator supports **multi-file configuration** - it merges multiple YAML files where:
- **Base configuration**: First file provides site metadata (colors, branding, etc.)
- **Navigation tabs**: All `navigation.tabs` from all files are combined in order
- **Icon support**: Tabs can specify icons with the `icon` field

**Schema validation is enabled by default** and validates against the Mintlify schema. Use `make check-missing` to find any documentation files that aren't included in the navigation.

### Making Changes
1. Edit `.mdx` files directly for content updates
2. Update YAML config files:
   - `common.yaml` for site metadata (colors, branding, global navigation)
   - `omni.yaml` for Omni-specific navigation tabs
   - Additional YAML files for other product tabs
3. Run `make docs.json` to regenerate the configuration
4. Add images to appropriate `images/` subdirectories
5. Use Docker setup above to preview changes locally

### Multi-File Configuration
The generator merges YAML files in the order specified:
```bash
# Using container (recommended)
docker run --rm -v $(PWD):/workspace -w /workspace ghcr.io/siderolabs/docs-gen:latest common.yaml omni.yaml additional-tabs.yaml

# Using local Go build
cd docs-gen && go run main.go ../common.yaml ../omni.yaml ../additional-tabs.yaml
```

Example tab with icon:
```yaml
navigation:
  tabs:
    - tab: "Product Name"
      icon: "/images/product.svg"  # or FontAwesome icon name
      groups:
        - group: "Getting Started"
          folder: "product/getting-started"
```

### Manual Page Configuration
The generator supports two approaches for page organization:

**Option 1: Automatic folder scanning (current default)**
```yaml
navigation:
  tabs:
    - tab: "Docs"
      groups:
        - group: "Getting Started" 
          folder: "docs/getting-started"
          order:  # Optional custom ordering
            - "intro.mdx"
            - "setup.mdx"
```

**Option 2: Manual page definition (NEW)**
```yaml
navigation:
  tabs:
    - tab: "Docs"
      groups:
        - group: "Security and Authentication"
          folder: "omni/security-and-authentication"  # Optional base path
          pages:
            - "authentication-and-authorization"
            - "how-to-manage-acls" 
            - "omni-kms-disk-encryption"
            - group: "Rotate Siderolink Join Token"
              pages:
                - "rotate-siderolink-join-token/rotate-siderolink-join-token"
            - group: "Using SAML With Omni"
              pages:
                - "using-saml-with-omni/auto-assign-roles-to-saml-users"
                - "using-saml-with-omni/configure-saml-and-acls"
                - "using-saml-with-omni/how-to-configure-entraid-for-omni" 
                - "using-saml-with-omni/overview"
```

**Key features of manual page configuration:**
- **Mixed content**: Can combine individual pages and subgroups
- **Nested subgroups**: Subgroups can contain other subgroups
- **Base path**: Optional `folder` field prepends to all page paths
- **Flexible structure**: Full control over navigation hierarchy
- **Backward compatible**: Falls back to folder scanning if no `pages` defined

### Content Guidelines
- Focus on practical, actionable guidance for Omni users
- Include screenshots for UI-based procedures
- Maintain consistency with existing documentation structure and tone
- All content relates to Kubernetes management, infrastructure setup, or security configuration

## Important Notes

- **Containerized generator**: The `docs.json` file is generated using a containerized tool (`ghcr.io/siderolabs/docs-gen`)
- **Multi-file configuration**: Generated from multiple YAML files - edit the YAML files, not the JSON directly
- **Automatic page discovery**: The generator automatically discovers all `.mdx` files in configured folders
- **CI/CD integration**: The container is built and published automatically via GitHub Actions
- **Mintlify hosting**: The site is hosted and built by Mintlify using the `docs.json` configuration
- **Static assets**: All images are committed to the repository in `images/` directories

## Generator Configuration

### Code Structure
- **`docs-gen/`**: Contains the Go generator code
  - `main.go`: Main generator logic
  - `go.mod`, `go.sum`: Go module files
- **YAML config files**: In root directory (e.g., `common.yaml`, `omni.yaml`)

### Configuration Files
The YAML config files control:
- **Site metadata**: name, colors, logos, banners
- **Schema validation**: URL for Mintlify JSON schema validation
- **Navigation structure**: tabs, groups, and folder mappings
- **Page ordering**: optional custom ordering within groups
- **Global navigation**: footer links, external anchors
- **Redirects**: URL redirects for moved or renamed pages

Each navigation group maps to a folder, and all `.mdx` files in that folder are automatically included. Subdirectories become nested groups.

### Schema Validation
The generator validates the output against the Mintlify schema by default. To skip validation (not recommended), use:
```bash
go run main.go --skip-validation common.yaml > docs.json
```

### Redirects
Add redirects to handle moved or renamed pages:
```yaml
redirects:
  - source: "/old-path"
    destination: "/new-path"
  - source: "/legacy-docs/*"
    destination: "/docs/$1"
```

Redirects support wildcards (`*`) and parameter substitution (`$1`, `$2`, etc.) as described in the [Mintlify redirects documentation](https://mintlify.com/docs/settings/broken-links#redirects).
