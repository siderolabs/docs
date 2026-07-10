# version-upgrade-gen

A Go tool that points the Talos documentation at a specific Talos release tag.

## What it does

You pass the exact target image tag with `--tag` (e.g. `v1.14.0-beta.0` or
`v1.14.0`). The tool reads the current tag from `TALOSCTL_IMAGE` in the Makefile
and derives everything else from the two tags:

- **Stage** (alpha / beta / stable) comes from the tag's suffix — there is no
  interactive prompt.
- **Folder** (`TALOS_VERSION`, e.g. `v1.14`) is the tag's `major.minor`. Moving a
  minor through alpha → beta → stable keeps the same folder; only a genuinely new
  minor changes it.

**Pre-release target (alpha/beta)**
- Rewrites `TALOSCTL_IMAGE` to the exact tag and `TALOS_VERSION` to the folder minor
- Updates (or appends) the versioned variables block in `custom-variables.mdx` with the tag
- Leaves the "latest stable" pointer (banner, canonical URLs, latest-stable block) untouched
- **Navigation differs by stage:**
  - **alpha** — leaves the `docs.json`/`check-missing` nav lists unchanged. The reference
    docs are still generated on disk, but the version stays out of the published nav.
  - **beta** — ensures `talos-vX.Y.yaml` is present at the bottom of the nav in all four
    Makefile targets so the version appears last in the dropdown. This is idempotent, so
    a beta correctly adds the entry that an earlier alpha of the same minor skipped.

**Stable target**
- Rewrites `TALOSCTL_IMAGE` to the exact tag and `TALOS_VERSION` to the folder minor
- Updates the "latest stable" block in `custom-variables.mdx` with the release's
  Kubernetes (from the `siderolabs/talos` release notes) and nvidia versions, and
  updates/creates the versioned block
- Only when the latest stable minor actually changes (read from the version warning
  banner, not from `TALOS_VERSION`): updates the banner, rewrites canonical URLs
  across older version docs, and moves `talos-vX.Y.yaml` to the top of the nav

**nvidia values (both paths)**
The `nvidia_container_toolkit_release` / `nvidia_driver_release` values are resolved
the same way everywhere: first from the `siderolabs/extensions` release notes for the
exact tag, and — if that release is missing, has no nvidia fields, or the fetch fails —
falling back to the current unversioned latest-stable values. They are therefore never
written empty, and a stable run no longer aborts when the extensions data lags behind.
For an existing versioned block, a pre-release only backfills nvidia values that are
currently empty, so good values from an earlier stable run are never overwritten.

The current image tag is matched in full, including any pre-release suffix, so a pin
like `v1.14.0-alpha.2` is replaced cleanly rather than by prefix.

## Usage

From the repo root, pass the target tag with `TAG=`:

```bash
# Published Docker container
make upgrade-talos-version TAG=v1.14.0-beta.0     # beta of the current minor
make upgrade-talos-version TAG=v1.14.0            # stable
make upgrade-talos-version TAG=v1.15.0-beta.0     # first beta of a new minor

# Local Go build
make upgrade-talos-version-local TAG=v1.14.0-beta.0
```

After the tool rewrites the pins, the Makefile targets automatically regenerate the
Talos reference docs (`make generate-talos-reference`), the changelog, and
`docs.json`, then tell you to run `make preview` to verify the result.

Running the binary directly:

```bash
go run . --workspace .. --tag v1.14.0-beta.0
```

## Tag validation

The `--tag` value must be `vMAJOR.MINOR.PATCH` with an optional `-alpha.N` /
`-beta.N` / `-rc.N` suffix (e.g. `v1.14.0`, `v1.14.0-beta.0`). Surrounding
whitespace is trimmed. Anything else — a missing `v`, a missing patch number, a
misspelled stage, etc. — is rejected up front, before any file is modified, so a
typo can never leave the repo in a half-applied state.

## Prerequisites

- A `talos-vX.Y.yaml` navigation file must already exist before running
- A `GITHUB_TOKEN` is used for the stable release GitHub lookups (and by the
  changelog step). The Makefile auto-fills it from `gh auth token` when it isn't
  already set in the environment, so no manual export is normally needed.

## Building the container

```bash
make build-version-upgrade-container
```

The container is automatically built and published to `ghcr.io/siderolabs/version-upgrade-gen:latest` when changes to this directory are merged into `main`.
