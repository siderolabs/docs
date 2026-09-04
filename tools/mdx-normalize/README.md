# mdx normalize

This tool cleans up generated Markdown/MDX so it renders correctly on Mintlify.

Generators such as `omnictl docs` and `talosctl docs`, and upstream files such
as the image-factory `configuration.md`, produce constructs that Mintlify (which
parses `.mdx` as MDX/JSX) does not handle. This tool normalizes them in place:

- **Tab-indented code blocks** are converted to fenced (```` ``` ````) code
  blocks. Mintlify does not support indented code blocks, so a bare `<` in a
  line like `source <(talosctl completion bash)` is otherwise read as JSX and
  breaks the build.
- **Tab-indented prose** in a command's "Synopsis" is left as a normal
  paragraph. Command examples are told apart from prose by their intro line:
  examples are introduced by a line ending in a colon (`...run:` or a
  `#### Linux:` heading), so a colon-introduced block is fenced and any other
  block is de-indented.
- **Tab-indented lists** nested under a list item (for example sub-bullets under
  `- For each node:`) are re-indented and kept as a list, rather than being
  fenced into a code block.
- With `--strip-hr`, standalone `---` horizontal-rule separators are removed
  (used for the image-factory reference, which puts a rule between every
  parameter).
- With `--escape-inline`, every `<` and `{` in prose is backslash-escaped
  (only inline code spans and already-escaped characters are skipped), so
  placeholders such as `<machine-id>` and expression-looking text such as
  `{"cni":"flannel"}` in CLI help do not break the build. This is **opt-in** and
  applied only to plain-markdown CLI help, because it must not run on files that
  contain real HTML/JSX — escaping would corrupt the `<table>` markup in the
  Talos configuration reference and the schema-generated Omni pages. Those files
  are normalized with escaping off, which leaves their HTML, MDX expressions
  (`{"<"}`) and comments (`{/* ... */}`) untouched.

A leading YAML frontmatter block is always preserved verbatim. `---` lines
inside a fenced code block are never touched.

Every transformation is idempotent: running the tool again on its own output is
a no-op.

## How to use

With one or more file arguments, each file is normalized in place:

```bash
go run . [--strip-hr] [--escape-inline] path/to/file.mdx [more.mdx ...]
```

With no argument (or `-`), it reads stdin and writes stdout. This filter mode is
how the container is used, so the file is never read or written through a bind
mount (which avoids Docker Desktop for macOS mount-consistency races):

```bash
docker run --rm -i ghcr.io/siderolabs/mdx-normalize:latest [--strip-hr] [--escape-inline] < in.mdx > out.mdx
```

It is normally invoked through the `normalize-doc` Make target (or its
`normalize-doc-local` counterpart). Each target picks the flags per file:
`--escape-inline` for the Omni CLI reference, `--strip-hr --escape-inline` for
the image-factory reference, and structural normalization only for everything
else (including the HTML-bearing Talos reference and schema-generated Omni
pages).

Run bare, it normalizes the changed reference `.mdx` files — anything git-dirty
under a `/reference/` path:

```bash
make normalize-doc         # container image
make normalize-doc-local   # local Go build
```

Pass `NORMALIZE_PATHS` (a space-separated list of files and/or directories;
directories are expanded to the `.mdx` under them) to normalize a specific set
instead of the changed files:

```bash
make normalize-doc NORMALIZE_PATHS="public/talos/v1.13/reference"
```

The reference-generation targets set `NORMALIZE_PATHS` to their own output and
call `normalize-doc` automatically, so each generator normalizes only what it
produced — regenerating one reference tree never rewrites another.

## Tests

```bash
go test ./...
```
