# CLAUDE.md

Guidance for AI agents (and humans) working in this repository.

## What this is

`gh-repos-hud` is a [`gh`](https://cli.github.com) CLI extension: a heads-up
display of repo health across the GitHub organizations you belong to plus your
personal repos. One Go binary with four front-ends sharing a single data core —
a bubbletea TUI (default), a loopback web dashboard (`serve`), `--plain` text,
and `--json`.

## Build, test, run

```sh
make build            # compile -> ./gh-repos-hud
make install          # rebuilds, then `gh extension install .` (run as `gh repos-hud`)
make test             # go test -race -count=1 ./...
make lint             # go vet + golangci-lint
make vuln             # govulncheck
gh repos-hud --demo   # synthetic dataset; needs no auth or network
```

After changing source, use `make install` — a bare `gh extension install .`
installs the *stale* prebuilt binary instead of recompiling. Confirm the build
with `gh repos-hud --version` (it stamps the commit SHA).

## Conventions (load-bearing)

- **No emojis anywhere.** Status markers are ASCII glyphs only: `[OK]` `[~]`
  `[!!]` `[?]`. This is a hard rule, enforced across every front-end.
- Authentication comes from `gh` via `cli/go-gh`. Never embed, log, or persist
  a token.
- Keep it `gofmt`-clean and `go vet`-clean; tests must pass under `-race`. CI
  enforces `gofmt`.
- Commit messages follow conventional-commit style: `feat:`, `fix:`, `perf:`,
  `docs:`, `chore:`.
- The web server (`serve`) binds to `127.0.0.1` only and rejects non-loopback
  `Host` headers. Keep it loopback-only.

## Architecture

- `internal/ghclient` — all GitHub I/O: one batched GraphQL query per owner plus
  per-repo REST scan probes, `FetchState` orchestration, the disk cache, and the
  `--demo` data.
- `internal/model` — front-end-agnostic types and the derived health/scan logic
  (`ComputeHealth`, the `ScanState` tri-state, etc.).
- `internal/tui`, `internal/web`, `cmd` — the four renderers, all over the same
  `model.State`.

## Security and releases

See [SECURITY.md](SECURITY.md). Auth is never embedded; the cache holds repo
metadata only. Releases are GPG-signed (`checksums.txt.sig`) and carry a
Sigstore build-provenance attestation — see "Verifying a release" in the README.
