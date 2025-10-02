# docs convert

This tool exists to convert documentation from markdown format to .mdx format for mintlify.
It takes talos reference documentation from `talosctl docs` and converts the files into `.mdx` file format with specific formatting for mintlify documentation hosting.

## How to use

This code is packaged as a container and is intended to be run manually when creating or updating reference documentation.

This command will use `talosctl` from a container and run the docs-convert against the output folder.
To run it manually you can run

```bash
go run main.go markdown-docs/ mdx-docs/
```

This will look for every `.md` file and change the extension to `.mdx` and apply some basic rules needed for mintlify.
It also ignores some unnecessary files like _index.md which were only used for `hugo`.

You can run these steps via the following make target.
This will temporarly write the markdown docs to `_out/docs` and move them into `public/talos/$VERSION/reference`.

```bash
make generate-talos-reference
```

If you want to generate a specific version of the documentation you will need to look up the `talosctl` image tag you want to use from [the talosctl repository](https://github.com/siderolabs/talos/pkgs/container/talosctl) and set the version where you want the documentation written.

An example for Talos 1.9

```bash
make generate-talos-reference \
  TALOSCTL_IMAGE="ghcr.io/siderolabs/talosctl:v1.9.5" \
  TALOS_VERSION=v1.9
```

## Development

If you need to add conversions or output to the code you can add them to main.go and run the conversion locally without a container via

```bash
make generate-talos-reference-local
```
