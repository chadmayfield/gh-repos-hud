#!/usr/bin/env bash
#
# Build and install gh-repos-hud as a local gh extension.
# After this, run it as: gh repos-hud
#
set -euo pipefail

cd "$(dirname "$0")"

echo "building gh-repos-hud..."
go build -trimpath -o gh-repos-hud .

# Reinstall cleanly if it's already present.
gh extension remove repos-hud >/dev/null 2>&1 || true
gh extension install .

echo "installed — run: gh repos-hud  (or: gh repos-hud serve)"
