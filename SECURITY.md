# Security Policy

## Reporting a vulnerability

Please report security issues **privately** via GitHub's private vulnerability
reporting: open the repository's **Security** tab and choose
**"Report a vulnerability"**. This keeps the report confidential until a fix is
available.

Please do not open a public issue for security problems.

When reporting, include where possible:

- affected version / commit,
- a description of the issue and its impact,
- reproduction steps or a proof of concept.

You can expect an initial acknowledgement within a few days. There is no bug
bounty; this is a personal open-source project maintained on a best-effort
basis.

## Security posture of this tool

`gh-repos-hud` is a local, read-only dashboard. A few properties are
load-bearing for its security and are good to keep in mind when assessing or
changing it:

- **No embedded credentials.** Authentication is sourced at runtime from the
  logged-in [`gh`](https://cli.github.com) CLI via the official
  `cli/go-gh` library. No token is ever read from the source, embedded in the
  binary, written to the cache, or logged.
- **Loopback-only web server.** `gh repos-hud serve` binds to `127.0.0.1`
  only and refuses non-loopback addresses; it is not intended to be exposed to
  a network.
- **Read-only.** The tool only issues GitHub read queries (repository health,
  alerts, PR metadata). It does not modify repositories.
- **Local cache.** The on-disk snapshot under the user cache directory contains
  repository metadata only — never tokens or secrets.

If you find a case where any of the above does not hold (for example, a code
path that could log or persist a token, or bind the server off-loopback),
please treat it as a security issue and report it as above.
