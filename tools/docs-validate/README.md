# docs-validate

A Go tool that validates all Talos documentation versions by cross-checking each `talos-vX.Y.yaml` navigation config against its corresponding `public/talos/vX.Y/` content directory.

## What it does

For every `talos-vX.Y.yaml` file found in the repo root, it checks both directions:

- **In yaml but file missing** — a page is listed in the navigation but the `.mdx` file doesn't exist in the content directory
- **File exists but not in yaml** — an `.mdx` file exists in the content directory but isn't listed in the navigation

It reports a per-version summary and exits with a non-zero status if any issues are found.

## Auto-insert (`--fix`)

With `--fix`, instead of only reporting, the tool inserts pages that exist on disk
but are missing from the nav — useful after a Talos upgrade regenerates the
reference docs. Each page is placed into the group that owns its folder (matched by
the folder the group's existing pages point at, not by the group's display title),
in alphabetical order, copying the surrounding indentation and quote style so the
diff is one line per page. A brand-new folder with no section yet gets a fresh
section named after the folder (with an acronym override map, e.g. `cri` → `CRI`,
title-case otherwise).

It is deliberately best-effort and **always exits 0** so it never blocks an upgrade:

- a folder owned by two groups (ambiguous) → warn and skip
- a folder it can't anchor a new section against → warn and skip

The plain (blocking) validation still runs afterwards as the correctness gate, so
anything `--fix` skipped is surfaced loudly.

```bash
# Report only (blocking)
make validate-docs-nav

# Insert missing pages, best-effort (non-blocking)
make sync-docs-nav
```

## Usage

```bash
# Run across all versions
make validate-docs-nav
```

Example output:

```
v1.13        OK
v1.12        OK
v1.11        1 issue(s)
  file exists but not in yaml: talos/v1.11/reference/configuration/cli.mdx
v1.10        OK
...

All versions OK
```

## When to use it

- **During alpha/beta scaffolding** — catch missing files or unlisted pages before the stable release
- **After any content changes** — confirm the nav and content stay in sync
- **Automatically** — runs at the end of every `make upgrade-talos-version` and `make upgrade-talos-version-local`
