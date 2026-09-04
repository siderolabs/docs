# canonical-gen

Makes sure every Talos documentation page declares a `canonical:` link in its
YAML frontmatter, naming the page that is currently authoritative for that
content.

Auto-generated pages (from `talosctl docs` or a version upgrade) arrive without
a canonical link, and older versioned pages must defer to the current version
rather than each claiming authority over the same content.

## Why this is not a path rewrite

Every Talos version lives in its own directory and every directory is
permanent. When the docs are restructured nothing is ever *moved*: v1.11 keeps
`networking/vip.mdx` forever, while v1.12 onwards has
`networking/advanced/vip.mdx`. The two paths are a correspondence between
diverged trees, not a rename — so neither the filesystem nor git history
records the relationship, and it has to be inferred.

## How a target is chosen

For a page at `public/talos/<own>/<relPath>.mdx`, in order:

0. `<relPath>` is a **per-release document** → point at the page's own version.
   Release notes are separate documents per release, not versions of one page:
   "What's New in Talos 1.6.0" is not an outdated rendering of the 1.14 page,
   and pointing the older at the newer would deindex content the newer page
   does not contain. Detected from a title that both *varies* across versions
   and *names a release* — a page maintained across releases keeps one title
   ("Virtual (shared) IP"), and a plain rewording ("Deploy First Workload")
   names no release. A false positive here costs a missed consolidation; a
   false negative destroys a page's indexing, so the rule is deliberately
   biased towards the former.
1. `<relPath>` exists in the current version → point there. This also covers a
   page that already *is* in the current version, which points at itself.
2. Otherwise, find the newest version that still has `<relPath>` and look at
   the pages born in the version right after it — the ones that appeared
   exactly when this page disappeared. Pick a successor from that pool by the
   first rule that yields **exactly one** candidate:
   - **a.** same file name (`networking/vip` → `networking/advanced/vip`)
   - **b.** the page became a directory holding a single page
   - **c.** file-name word overlap (`wireguard-network` → `wireguard`)
   - **d.** body similarity ≥ 0.55 (`verifying-images` → `source-talos-images`)

   Rules a–c never look at page content, so they survive a full rewrite; rule d
   catches renames that preserved the text. A winner is resolved recursively, so
   a page that moves twice still lands on its final home.
3. If no rule wins — the page was split into several, or simply dropped — point
   at the newest version that still has `<relPath>`. That is the newest
   surviving copy of this exact content, it always exists, and it consolidates
   every older copy onto one URL.

Nothing is guessed: a rule must produce a single unambiguous candidate or it is
skipped. Pages that reach step 3 are reported, so a genuine deletion stays
distinguishable from a restructure the rules could not follow:

```
canonical-gen: 5 page(s) have no equivalent in the current version;
  older copies now point at the newest version that still has them:
    networking/advanced-networking -> v1.11
    networking/device-selector -> v1.11
    ...
```

An empty report is the steady state, which is what makes a non-empty one worth
reading. Per-release documents are reported separately, so the exemption is
visible rather than silent:

```
canonical-gen: 1 page path(s) are per-release documents;
  every version's copy is authoritative for its own release:
    getting-started/what's-new-in-talos
```

## The canonical URL

```
https://docs.siderolabs.com/talos/<version>/<relPath>
```

`<version>` is normally `export const version` from
`public/snippets/custom-variables.mdx`, and `<relPath>` the page path below the
version directory. The link is inserted as the last field of the existing
frontmatter block, or rewritten in place when a `canonical:` line already
exists. Everything else in the file — including the rest of the frontmatter and
its field order — is preserved verbatim.

## Usage

```bash
go run . [--variables path/to/custom-variables.mdx] [--version vX.Y] [--check] [path ...]
```

Each path is a file or a directory to walk for `.mdx` files; with no path the
tool processes `public/talos`. Files outside a `public/talos/<version>/` tree
are skipped. With `--check` nothing is written and the tool exits non-zero if
any file would change.

Version directories are always scanned in full, whichever paths are given, so
running against a handful of files yields the same answers for those files as
running against the whole tree.

Normally invoked through Make:

```bash
make canonical-links              # every Talos page (container)
make canonical-links-local        # every Talos page (local Go build)
make canonical-links-check        # report only, writes nothing
make canonical-changed            # changed .mdx files only (container)
make canonical-changed-local      # changed .mdx files only (local Go build)
```

The `*-changed` targets take their base revision from `CANONICAL_BASE`
(default `HEAD`) and are the ones to use after regenerating reference docs,
which rewrites ~90 pages and drops their frontmatter.

## Tests

```bash
go test ./...
```
