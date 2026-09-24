## Omni Helm Chart Gen

This utility generates the Omni Helm chart reference page (`public/omni/reference/helm-chart.mdx`) from the chart README in the Omni repository.

The README is the source of truth for the chart and is versioned with it. This tool copies it into the docs site and converts the GitHub-flavored Markdown that Mintlify cannot render as MDX:

- Drops the H1, the version badges, the homepage line, and the Maintainers section.
- Converts GitHub alerts (`> [!NOTE]`, `> [!WARNING]`, ...) to Mintlify callouts.
- Converts bare autolinks (`<https://...>`, `<user@example.com>`) to Markdown links, since MDX parses a bare `<` as JSX.
- Rewrites the relative `values.yaml` links to the chart directory on GitHub.

Code blocks are copied unchanged.

### Usage

Pass a tag or branch of `siderolabs/omni`. With no argument, the tool uses the latest Omni release, so the page matches the chart published to `ghcr.io`:

```bash
# Latest Omni release
go run . > ../../public/omni/reference/helm-chart.mdx

# A specific release
go run . v1.12.2 > ../../public/omni/reference/helm-chart.mdx

# A local README (the ref is still used for links)
go run . -readme /path/to/README.md v1.12.2 > ../../public/omni/reference/helm-chart.mdx
```

Or use the Makefile target from the repo root, which also normalizes the page:

```bash
make generate-omni-helm-chart-reference
```

To pin a release:

```bash
make generate-omni-helm-chart-reference OMNI_HELM_CHART_REF=v1.12.2
```

### Tests

```bash
go test ./...
```
