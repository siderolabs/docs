# omni-cli-gen

This is a **docs-tooling helper**, not an official `omnictl` distribution. It
packages a pinned [`omnictl`](https://github.com/siderolabs/omni) binary in a
container purely so the Omni CLI reference (`public/omni/reference/cli.mdx`) can
be generated without installing `omnictl` locally.

There is no official `omnictl` image, so this Dockerfile downloads a pinned
release binary from the `siderolabs/omni` releases and ships it in an Alpine
image. This mirrors how the Talos docs consume a pinned `talosctl` image — the
difference is that image is docs-scoped and named accordingly.

## How to use

The image is published as `ghcr.io/siderolabs/omni-cli-gen` and used by the
`generate-omni-cli-reference` Make target. To avoid writing generated docs into
a bind-mounted host directory (which is unreliable on Docker Desktop for macOS),
`omnictl` runs inside the container and the result is streamed to stdout:

```bash
docker run --rm --entrypoint /bin/sh ghcr.io/siderolabs/omni-cli-gen:latest \
  -c 'omnictl docs /tmp >/dev/null 2>&1 && cat /tmp/cli.md' > cli.md
```

The default entrypoint is `omnictl`, so the image can also be used as the CLI
directly (e.g. `docker run --rm ghcr.io/siderolabs/omni-cli-gen:latest --version`).

## Pinning a version

The omnictl version is set by the `OMNICTL_VERSION` build argument (default
`v1.9.0`). Override it at build time:

```bash
docker build --build-arg OMNICTL_VERSION=v1.9.0 -t ghcr.io/siderolabs/omni-cli-gen:latest .
```
