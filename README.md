# gh-repos-hud

A heads-up display of repo health across every GitHub organization you belong
to (plus your personal repos), as a [`gh`](https://cli.github.com) extension.

One glance shows, per repo, grouped by org:

- **Health** rollup (`[OK]` / `[~]` / `[!!]`) from CI + alerts + undeployed changes
- **CI** status of the latest run on the default branch
- current **short SHA** and **latest version tag / release**
- **undeployed changes** — commits on the default branch since the last tag
- **Dependabot** alerts by severity, plus **code-scanning** and **secret-scanning** counts
- **scanning coverage** (enabled?) and active paid **GHAS** spend per org
- open **PRs** split into Dependabot-vs-human, with mergeable / CI-green counts

Auth is sourced from `gh` — **no token is ever embedded or stored** by this tool.

## Install

```sh
gh extension install chadmayfield/gh-repos-hud
```

## Usage

```sh
gh repos-hud                      # interactive TUI (default)
gh repos-hud serve --port 8787    # local web dashboard at http://127.0.0.1:8787
gh repos-hud --json               # machine-readable snapshot
gh repos-hud --org acme       # limit to specific orgs (repeatable)
gh repos-hud --only-attention     # only repos needing attention (non-green)
```

Requires a `gh` login with at least `repo` and `read:org` scopes;
`security_events` / `admin:org` enable the code/secret-scanning and GHAS-billing
columns (missing scopes degrade those cells to `?` rather than failing).

## Build from source

```sh
make build      # -> ./gh-repos-hud
make install    # gh extension install . (run as `gh repos-hud`)
make test lint vuln
```
