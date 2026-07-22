# style guide checker

A small linter that checks SideroLabs documentation (`.mdx`) against the
mechanically-checkable parts of the [documentation style guide][guide]. It is the
repo's single style tool (it replaced Vale).

## What it checks

| Rule | Finding | Level |
|------|---------|-------|
| Titles — title case | `title/title-case` — a significant word in the page `title` is left lowercase | warning |
| Headings — sentence case | `headings/sentence-case` — a word is over-capitalized (Title Case) in a heading | warning |
| Headings — no stacked headings | `headings/stacked` — a heading directly follows another with no sentence between | warning |
| Headings — step down one level at a time | `headings/skipped-level` — level jumps by more than one (e.g. `##` → `####`) | warning |
| Headings — avoid H1 in the body | `headings/avoid-h1` — a body `#` heading (the title is the H1) | warning |
| Headings — blank line around them | `headings/blank-line` — a heading not surrounded by blank lines | warning |
| Code — language hint required | `code/no-language` — a fenced block has no language | warning |
| Code — no shell prompts | `code/shell-prompt` — a `$ ` prompt in a copy-pasteable command | **error** |
| Code — portable commands | `code/sed` — uses `sed`, which differs between BSD and GNU | warning |
| Links — descriptive text | `links/non-descriptive` — text like "click here" / "here" | warning |
| Images — kebab-case filenames | `images/filename` — a referenced image or committed image file is not kebab-case | warning |

The page's frontmatter `title` is treated as the H1, so the first body heading is
expected to be `##`. Content inside fenced code blocks is ignored by the heading,
link, and image rules.

### The exceptions list (case checks)

Sentence/title case can't tell a proper noun from an over-capitalized word on its
own, so `exceptions.txt` holds the words allowed to keep their capitalization
(`Talos`, `Omni`, `Kubernetes`, CLI tool names, …). You only need to list plain
`Capitalized` words — **ALL-CAPS acronyms** (`SAML`, `YAML`) and **CamelCase
names** (`KubeSpan`, `SideroLabs`) are auto-allowed. Add a word whenever the
checker wrongly flags a real proper noun, then rebuild the image.

### Not checked (on purpose)

- **UI text in bold** and **numbered vs. unordered lists** — these need to know
  *intent* (is this word a button? is this list sequential?) and can't be
  detected mechanically without heavy false positives.
- **Personal data in screenshots** — image *contents* can't be inspected here;
  review those manually.

## Usage

```bash
# Lint everything under public/ (default)
go run .

# Lint a single file or directory
go run . ../public/omni/getting-started

# Fail on warnings too, not just errors
go run . -strict ../public

# Emit GitHub Actions annotations
go run . -format github ../public
```

By default the run exits non-zero only when there are **error**-level findings
(a shell prompt in a command). Use `-strict` to also fail on warnings.

### Via the Makefile

From the repo root:

```bash
make style-check                    # lint all of public/ (Docker, like the other tools)
make style-check DOC=public/omni    # lint a subtree
make style-check-local              # same, but using a local Go build (no Docker)
make style-check-changed            # lint only changed .mdx (Docker)

# extra flags for any of the above:
make style-check STYLE_CHECK_ARGS="-strict"          # fail on warnings too
make style-check-local STYLE_CHECK_ARGS="-format github"
```

### In CI

`style-check-changed` only checks the files a change touches, so a PR is gated on
its own new content instead of the whole (legacy) `public/` tree. Point it at the
PR base branch:

```bash
make style-check-changed STYLE_CHECK_BASE=origin/main STYLE_CHECK_ARGS="-format github"
```

Locally the default base is `HEAD`, which catches your uncommitted working-tree
edits. In CI, `HEAD` would be empty (everything is committed), so set
`STYLE_CHECK_BASE` to the base branch you are merging into.

`make style-check` runs the containerized tool, matching `make docs.json`,
`make normalize-doc`, etc. Until the image is published by CI, build it once so
the local `docker` run has something to use:

```bash
make build-style-check-container
```

`pull_if_missing` then reuses that local image; after CI publishes
`ghcr.io/siderolabs/style-guide-checker`, other machines pull it automatically.

[guide]: ../contributing-guides
