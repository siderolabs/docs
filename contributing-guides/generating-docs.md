# Generating docs from upstream sources

Some pages in this repo are **generated from upstream sources**, not written by hand. Regenerate them when the upstream tool or schema changes rather than editing the pages directly. This covers the **Talos** and **Omni** reference docs, as well as the auto-generated changelog.

## What's generated

Each page below is built from an upstream tool or file. The table shows where the page lives and the command that regenerates it; the sections that follow explain how each one works.

| Reference | Output | Source | Command |
| --- | --- | --- | --- |
| Talos config | `public/talos/<version>/reference/configuration/` | `talosctl docs` | `make generate-talos-reference` |
| Omni CLI | `public/omni/reference/cli.mdx` | `omnictl docs` | `make generate-omni-cli-reference` |
| Omni config | `public/omni/reference/omni-configuration.mdx` | Omni config JSON schema | `make generate-omni-config-reference` |
| Image Factory | `public/omni/reference/image-factory-configuration.mdx` | image-factory [`configuration.md`](https://github.com/siderolabs/image-factory/blob/main/docs/configuration.md) | `make generate-omni-image-factory-reference` |
| Changelog | `public/changelog.mdx` | GitHub releases | `make changelog` |

## Prerequisites

- **Docker** — the generators run as containers. Talos generation always needs Docker (it runs `talosctl` from a container). For Omni, the default targets are containerized too.
- **Network access** — images are pulled and sources are fetched from GitHub.

Every target also has a **`-local`** variant that runs the Go tools directly instead of pulling their container images (see each section). Those require **Go** installed — and, for the Omni CLI reference, a local **`omnictl`**.

> The Omni tool images (`omni-cli-gen`, `omni-config-gen`, `mdx-normalize`) are published to `ghcr.io` by CI when this 
> merges to `main`. Before that, build them once locally with the `build-*-container` targets — the targets use a 
> locally-present image before trying to pull.

## Talos reference

`make generate-talos-reference` regenerates the Talos machine-configuration
reference into `public/talos/<version>/reference/configuration/`.

How it works:

1. `talosctl docs` runs from the official `ghcr.io/siderolabs/talosctl` image
   and writes Markdown into a temporary `_out/docs` directory.
2. The [`docs-convert`](../docs-convert/README.md) tool converts that Markdown
   into Mintlify-flavored MDX in the output directory.

Targets:

```bash
make generate-talos-reference        # talosctl (container) + docs-convert (container)
make generate-talos-reference-local  # talosctl (container) + docs-convert (local go run)
```

Both variants run `talosctl` from a container, so **Docker is required either way**; the `-local` variant only runs the conversion step with local Go.

**Versioning:** the output directory is chosen by `TALOS_VERSION` in the Makefile (e.g. `v1.14`), and the talosctl image is pinned by `TALOSCTL_IMAGE`. Older versions already committed under `public/talos/` are not touched.

## Omni reference

`make generate-omni-reference` regenerates all three Omni reference pages; each page also has its own target if you only need one:

```bash
make generate-omni-reference                    # all three
make generate-omni-cli-reference                # cli.mdx
make generate-omni-config-reference             # omni-configuration.mdx
make generate-omni-image-factory-reference      # image-factory-configuration.mdx
```

How each page is built:

- **CLI** — `omnictl docs` runs from the `omni-cli-gen` container; its output is combined with a fixed frontmatter block and cleaned up.
- **Config** — the `omni-config-gen` tool reads the Omni config JSON schema and writes the whole page (frontmatter included).
- **Image Factory** — the upstream `configuration.md` is downloaded from GitHub, combined with a fixed frontmatter block, and cleaned up.

Each page is rebuilt entirely from its upstream source and written atomically (temp file, then rename), so a page is fully restored even if it was **deleted or emptied**, and a failure never blanks an existing page. The frontmatter (title/description) is defined in the Makefile, so it is always present — change it there, not in the generated file.

### `-local` variants

Each target has a `-local` counterpart that runs the Go tools with `go run`, using a local `omnictl` for the CLI reference, instead of the container images:

```bash
make generate-omni-reference-local
make generate-omni-cli-reference-local
make generate-omni-config-reference-local
make generate-omni-image-factory-reference-local
make normalize-doc-local
```

### Building the Omni tool images locally

Until the images are published to `ghcr.io`, build them locally so the container targets have an image to use:

```bash
make build-omni-cli-gen-container
make build-omni-config-gen-container
make build-mdx-normalize-container
```

### Normalization

Raw generator output contains constructs that Mintlify (which parses `.mdx` as MDX/JSX) does not render correctly. The CLI and Image Factory pages are cleaned up by [`mdx-normalize`](../mdx-normalize/README.md), which:

- converts tab-indented code blocks into fenced code blocks,
- keeps tab-indented "Synopsis" prose as normal paragraphs,
- backslash-escapes inline placeholders such as `<machine-id>` so MDX does not read them as JSX,
- strips the `---` horizontal-rule separators from the Image Factory page.

The configuration page is already valid MDX and is not normalized.

## Changelog

The changelog at `public/changelog.mdx` is generated from the projects' GitHub releases rather than written by hand.

`make changelog` runs the `changelog-gen` tool, which reads the GitHub releases and writes the page.

```bash
make changelog        # changelog-gen (container)
make changelog-local  # changelog-gen (local go run)
```

## Tools

Each generator lives in its own directory with a `Dockerfile`, and most also have a README with tool-specific detail. They are all this repo's own tooling; the one external dependency is called out below the table.

| Directory | Role | Image ownership |
| --- | --- | --- |
| [`docs-convert/`](../docs-convert/README.md) | Talos Markdown → MDX | this repo |
| [`omni-cli-gen/`](../omni-cli-gen/README.md) | packages a pinned `omnictl` | this repo (docs helper) |
| [`omni-config-gen/`](../omni-config-gen/README.md) | Omni config reference from the JSON schema | this repo |
| [`mdx-normalize/`](../mdx-normalize/README.md) | normalizes generated MDX for Mintlify | this repo |
| `changelog-gen/` | changelog from GitHub releases | this repo |

`talosctl` is consumed from Siderolabs' **official** image; the rest are docs-repo tooling.

## A note on containers and bind mounts

The Omni container targets stream through stdin/stdout instead of writing generated files into a bind-mounted host directory. This sidesteps Docker Desktop for macOS bind-mount consistency issues that can otherwise produce truncated or stale files.
