# frontmatter-check

A Go tool that verifies every `.mdx` page under `public/omni`, `public/talos`, and
`public/kubernetes-guides` carries the frontmatter fields its section requires.

## What it does

- **Talos pages** (`public/talos/**`) must have `title`, `description`, and `canonical`
- **Omni pages** (`public/omni/**`) and **Kubernetes guides** (`public/kubernetes-guides/**`)
  must have `title` and `description`

Any page missing a required field is reported, and the tool exits with a non-zero
status if any issues are found.

Every `.mdx` file under `public/` must be accounted for. A file either belongs to
one of the sections above and is checked, or it appears in the `exempt` list in
`main.go` as deliberately not page content — currently `public/changelog.mdx`
(generated) and `public/snippets` (shared `export const` variables). Exempt files
are named in the summary so the "Checked N" count can always be reconciled
against the number of files handed in.

A file matching neither is an **error**, not a silent skip:

```
2 file(s) belong to no known section:
  public/hypervisor/getting-started/install.mdx
  public/hypervisor/getting-started/overview.mdx
```

This is deliberate. A new content tree that nothing knows about would otherwise
be skipped in full and report green, leaving a whole product's pages unenforced
with no signal that anything was ignored. Registering the tree in `sections` or
`exempt` is a one-line change; discovering months later that it was never checked
is not.

Frontmatter is parsed as real YAML (`gopkg.in/yaml.v3`), so block scalars
(`description: |` followed by indented lines) resolve to their actual text —
an empty block scalar is correctly seen as missing, rather than a naive
line scan mistaking the `|` indicator itself for the value.

## Usage

```bash
# Check every .mdx page under public/
make check-frontmatter

# Check only changed .mdx files (used by CI). Base: FRONTMATTER_CHECK_BASE (default HEAD)
make check-frontmatter-changed
```

Example output:

```
public/talos/v1.14/security/cert-management.mdx: missing "description"
public/talos/v1.14/configure-your-talos-cluster/system-configuration/kubeprism.mdx: missing "canonical"

Checked 1705 file(s): 2 issue(s)
Skipped 2 exempt file(s): public/changelog.mdx, public/snippets/custom-variables.mdx
```

## Why a changed-files variant

`check-frontmatter-changed` diffs against a base ref and checks only the `.mdx`
files a PR actually touches, mirroring `style-check-changed`. This is the variant
`docs-ci.yaml` runs.

The scoping is per *file*, not per *line*. The tool validates the entire
frontmatter block of every file in the diff, so a PR that edits a page for an
unrelated reason still has to satisfy whatever that page is missing. Pages a PR
never opens are left alone, but "changed files" is not "changed lines" — so this
variant narrows the blast radius of the legacy backlog without eliminating it.
Running it blocking is only safe once the full scan (`make check-frontmatter`)
is clean.
