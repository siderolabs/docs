# docs-validate

A Go tool that validates all Talos documentation versions by cross-checking each `talos-vX.Y.yaml` navigation config against its corresponding `public/talos/vX.Y/` content directory.

## What it does

For every `talos-vX.Y.yaml` file found in the repo root, it checks both directions:

- **In yaml but file missing** — a page is listed in the navigation but the `.mdx` file doesn't exist in the content directory
- **File exists but not in yaml** — an `.mdx` file exists in the content directory but isn't listed in the navigation

It reports a per-version summary and exits with a non-zero status if any issues are found.

## Usage

```bash
# Run across all versions
make validate-talos-docs
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
