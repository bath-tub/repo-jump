#!/usr/bin/env bash
# repo-jump installer: builds the `rj` binary, puts it on your PATH, then runs
# the interactive setup wizard (gh auth -> pick org -> index -> keybinding).
set -euo pipefail

cd "$(dirname "$0")"

command -v go >/dev/null 2>&1 || {
	echo "error: Go is required to build rj — install from https://go.dev/dl" >&2
	exit 1
}

echo "Building rj…"
go build -o rj .

BIN_DIR="${REPO_JUMP_BIN:-$HOME/.local/bin}"
mkdir -p "$BIN_DIR"
cp rj "$BIN_DIR/rj"
echo "Installed rj to $BIN_DIR/rj"

case ":$PATH:" in
	*":$BIN_DIR:"*) ;;
	*) echo "warning: $BIN_DIR is not on your PATH — add it so \`rj\` is runnable" >&2 ;;
esac

# Hand off to the TUI wizard for the interactive parts.
exec "$BIN_DIR/rj" setup
