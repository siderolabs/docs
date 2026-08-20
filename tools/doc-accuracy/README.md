# doc-accuracy

An AI-powered accuracy check for the documentation. It answers a question the
mechanical checks can't: **"if a reader followed this page, would it harm them —
and is what we say still true?"**

The most dangerous documentation bug is a command that **succeeds and does harm
anyway** — for example a stateful service (a database, etcd, a registry) started
without the volume mount that persists its data to the host. The container runs
with no error, appears to work, and then silently loses everything the next time
it is recreated. A flag validator scoped to `talosctl`/`omnictl` would never
catch that: it can be a plain `docker` command, and the command is valid. The
only thing that catches that class of bug is a reviewer that reads the snippet
and reasons about its blast radius — so this runs the `claude` CLI headless as a
documentation reviewer.

## What it looks for

Beyond ordinary "wrong flag / wrong value / false claim" mistakes, it prioritizes
**harm**: commands that succeed but lose data (a stateful service run without its
persistence mount), destructive or irreversible operations (`rm -rf`, `dd`,
`docker volume rm`, `kubectl delete`, `git push --force`), removed safeguards,
and security downgrades (disabling TLS/auth, `chmod 777`, leaked secrets). It
applies this to **every** command in a snippet, not just the Sidero CLIs.

## How it works

The Go program (`main.go`) gathers the `.mdx` files to review, then runs
`claude -p` with the instructions embedded from `reviewer-prompt.md`. In the
default "changed" mode it also feeds the reviewer the `git diff`, so it focuses
on the exact lines you edited — the highest-risk place for a dropped flag or
mount. The reviewer may read repo files and `WebFetch` the upstream source of
truth to confirm a command or value:

- Talos / `talosctl` — <https://github.com/siderolabs/talos>
- Omni / `omnictl` — <https://github.com/siderolabs/omni>
- Extensions — <https://github.com/siderolabs/extensions>
- Discovery service — <https://github.com/siderolabs/discovery-service>

It runs read-only (Edit/Write are disallowed): it **reports**, it does not
change your docs. The full report is also written to
`_out/doc-accuracy-report.md`.

## Usage

```bash
# Review the .mdx files you've changed vs. HEAD (default)
make check-doc-accuracy

# Review one specific file (great for testing / spot checks)
make check-doc-accuracy DOC=public/omni/self-hosted/run-omni-on-prem.mdx

# Review changed files vs. a branch (e.g. before opening a PR)
make check-doc-accuracy DOC_ACCURACY_BASE=origin/main

# Review the entire public/ docs tree (slow)
make check-doc-accuracy-all
```

You can also run it directly:

```bash
cd tools/doc-accuracy
go run . -workspace ../.. public/omni/overview/what-is-omni.mdx   # one or more files
go run . -workspace ../.. -base origin/main                       # changed mode
go run . -workspace ../.. -all                                    # whole tree
```

## Exit codes

The `doc-accuracy` binary exits:

- `0` — reviewed, no critical issues (or nothing to review).
- `1` — reviewed, found critical issues (a snippet would harm the reader, break,
  or a claim is false).
- `2` — could not run (no `claude` on PATH, a git error, no verdict produced).

Note that both `go run` and `make` collapse any non-zero exit to a generic
failure (`go run` reports exit 1 for both 1 and 2; GNU make then exits 2 for any
failed recipe). So for gating, treat **non-zero as "failed"** — the run's
plain-English last line ("❌ … FAILED" vs. "Error: 'claude' CLI not found …")
tells you which case you're in. To branch on the exact code, run a built binary
(`go build` in this directory) rather than `go run`/`make`.

## Requirements

Go (to build/run the tool) and the [`claude` CLI](https://claude.com/claude-code),
signed in. Unlike the other tools in `tools/`, this one has **no container** — a
model call can't be containerized here — so it always runs as a local Go program.
Because it uses a model, its findings are advisory and can vary run to run — treat
it as a sharp reviewer, not a deterministic gate.

## Development

```bash
make test-doc-accuracy   # unit tests for file collection, verdict parsing, prompt assembly
```

It is a standard Go module, so `make code-review` lints it and `make test-all`
runs its tests along with the rest.

## Tuning the reviewer

Everything the reviewer looks for lives in `reviewer-prompt.md` (embedded into
the binary at build time) — edit that to add checks, tighten severity, or point
at more upstream files. Override the model with `DOC_ACCURACY_MODEL` (e.g. a
faster model for quick local passes).
