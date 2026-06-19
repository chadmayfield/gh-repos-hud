# Contributing

Thanks for your interest in improving `gh-repos-hud`. This is a small,
best-effort open-source project; issues and pull requests are welcome.

## Getting started

You need Go (the version is pinned in `go.mod`) and the
[`gh`](https://cli.github.com) CLI logged in (`gh auth login`).

```sh
make build      # compile -> ./gh-repos-hud
make test       # go test -race -count=1 ./...
make lint       # go vet + golangci-lint
make vuln       # govulncheck
make install    # install as a local gh extension (run as `gh repos-hud`)
```

Run `make install` after source changes — a plain `gh extension install .`
copies the existing prebuilt binary instead of rebuilding. Confirm with
`gh repos-hud --version`.

## Before opening a pull request

- Target the `main` branch.
- Make sure `make test lint` passes; CI runs the same checks and must be green.
- Keep changes focused; describe the motivation in the PR.
- Add or update tests for behavior changes (`httptest` fixtures for the
  GitHub client, table tests for derived model logic).

## Conventions

- **No emojis anywhere** — status markers are ASCII glyphs only
  (`[OK]` / `[~]` / `[!!]` / `[?]`). This is a hard project rule.
- Follow the surrounding Go style; run `gofmt` (or `make lint`).
- Commit messages follow a conventional-commit style:
  `feat:`, `fix:`, `perf:`, `docs:`, optionally scoped (e.g. `fix(cache): ...`).
- Never embed or log credentials. Authentication must continue to come from
  `gh` via `cli/go-gh`; see [SECURITY.md](SECURITY.md).

## Reporting bugs

Open an issue with the command you ran, what you expected, and what happened
(`HUD_DEBUG=1` enables debug logging to stderr). For security issues, do **not**
open a public issue — see [SECURITY.md](SECURITY.md).
