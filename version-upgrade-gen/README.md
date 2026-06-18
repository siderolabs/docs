# version-upgrade-gen

A Go tool that automates upgrading the Talos documentation to the next version.

## What it does

When run, it prompts you to select a release type and then:

**Beta release**
- Updates the versioned variables block in `custom-variables.mdx` with the new beta tag (e.g. `v1.14.0-beta.0`)
- Adds the new `talos-vX.Y.yaml` at the bottom of the nav in all four Makefile targets so the beta version appears last in the version dropdown

**Stable release**
- Updates the "latest stable" block in `custom-variables.mdx` with the new version's Kubernetes and nvidia versions (fetched from GitHub)
- Updates or creates the versioned variables block in `custom-variables.mdx`
- Updates the version warning banner to point to the new latest version
- Updates all canonical URLs across older version docs to point to the new version
- Moves the new `talos-vX.Y.yaml` to the top of the nav in all four Makefile targets
- Updates `TALOSCTL_IMAGE` and `TALOS_VERSION` in the Makefile

## Usage

From the repo root:

```bash
# Using local Go build
make upgrade-talos-version-local

# Using published Docker container
make upgrade-talos-version
```

Both commands will prompt you interactively:

```
What type of release is this?
  [1] Beta   - e.g v1.14.0-beta.0
  [2] Stable - e.g v1.14.0

Enter your choice (1 or 2):
```

After the tool runs, it automatically regenerates the changelog and `docs.json`, then tells you to run `make preview` to verify the result.

## Prerequisites

- A `talos-vX.Y.yaml` navigation file must already exist before running in either mode
- For stable releases, a `GITHUB_TOKEN` environment variable is recommended to avoid GitHub API rate limits

## Building the container

```bash
make build-version-upgrade-container
```

The container is automatically built and published to `ghcr.io/siderolabs/version-upgrade-gen:latest` when changes to this directory are merged into `main`.
