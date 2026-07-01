# mdx normalize

This tool cleans up generated Markdown/MDX so it renders correctly on Mintlify.

Generators such as `omnictl docs`, and upstream files such as the image-factory
`configuration.md`, produce constructs that Mintlify (which parses `.mdx` as
MDX/JSX) does not handle. This tool normalizes them in place:

- **Tab-indented code blocks** are converted to fenced (```` ``` ````) code
  blocks. Mintlify does not support indented code blocks, so a bare `<` in a
  line like `source <(omnictl completion bash)` is otherwise read as JSX and
  breaks the build.
- **Tab-indented prose** in a command's "Synopsis" is left as a normal
  paragraph. Command examples are told apart from prose by their intro line:
  examples are introduced by a line ending in a colon (`...run:` or a
  `#### Linux:` heading), so a colon-introduced block is fenced and any other
  block is de-indented.
- With `--strip-hr`, standalone `---` horizontal-rule separators are removed
  (used for the image-factory reference, which puts a rule between every
  parameter).

A leading YAML frontmatter block is always preserved verbatim. `---` lines
inside a fenced code block are never touched.

## How to use

With a file argument, the file is normalized in place:

```bash
go run . [--strip-hr] path/to/file.mdx
```

With no argument (or `-`), it reads stdin and writes stdout. This filter mode is
how the container is used, so the file is never read or written through a bind
mount (which avoids Docker Desktop for macOS mount-consistency races):

```bash
docker run --rm -i ghcr.io/siderolabs/mdx-normalize:latest [--strip-hr] < in.mdx > out.mdx
```

It is normally invoked through the `normalize-doc` Make target, which the
reference-generation targets call automatically:

```bash
make normalize-doc FILE=public/omni/reference/cli.mdx
make normalize-doc FILE=public/omni/reference/image-factory-configuration.mdx STRIP_HR=1
```

## Tests

```bash
go test ./...
```
