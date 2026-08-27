#!/bin/sh
set -eu

usage() {
  cat <<'USAGE'
Usage: ./install.sh [options]

Build and install the sima CLI from this source checkout.

Installation is binary-only by default. Project setup is optional and runs
through the installed Go CLI (`sima setup`) when explicitly requested.

Options:
  --bin-dir DIR       Install directory for the sima binary (default: $HOME/.local/bin)
  --setup DIR         Optionally run `sima setup --path DIR` after installing
  --project DIR       Alias for --setup DIR, kept for existing docs/scripts
  --backend MODE      Backend mode for optional setup: auto, claude, codex, none (default: auto)
  --help              Show this help

Examples:
  ./install.sh
  ./install.sh --bin-dir ~/bin
  ./install.sh --setup /path/to/project
  ./install.sh --setup . --backend none
USAGE
}

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

expand_path() {
  case "$1" in
    ~) printf '%s\n' "$HOME" ;;
    ~/*) printf '%s/%s\n' "$HOME" "${1#~/}" ;;
    *) printf '%s\n' "$1" ;;
  esac
}

BIN_DIR="${SIMA_INSTALL_DIR:-$HOME/.local/bin}"
SETUP_DIR=""
BACKEND="auto"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --bin-dir)
      [ "$#" -ge 2 ] || fail "--bin-dir requires a value"
      BIN_DIR="$2"
      shift 2
      ;;
    --setup|--project)
      [ "$#" -ge 2 ] || fail "$1 requires a value"
      SETUP_DIR="$2"
      shift 2
      ;;
    --backend)
      [ "$#" -ge 2 ] || fail "--backend requires a value"
      BACKEND="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      fail "unknown option: $1"
      ;;
  esac
done

case "$BACKEND" in
  auto|claude|codex|none) ;;
  *) fail "--backend must be one of: auto, claude, codex, none" ;;
esac

command -v go >/dev/null 2>&1 || fail "Go is required but was not found in PATH"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$SCRIPT_DIR"
[ -f "go.mod" ] || fail "install.sh must be run from a SIMA source checkout"

BIN_DIR=$(expand_path "$BIN_DIR")
mkdir -p "$BIN_DIR"

TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t sima-install)
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

log "Building sima..."
go build -o "$TMP_DIR/sima" ./cmd/sima
install -m 0755 "$TMP_DIR/sima" "$BIN_DIR/sima"
log "Installed: $BIN_DIR/sima"

if ! command -v sima >/dev/null 2>&1; then
  case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *)
      log "Note: $BIN_DIR is not on PATH. Add it or run SIMA as: $BIN_DIR/sima"
      ;;
  esac
fi

if [ -n "$SETUP_DIR" ]; then
  SETUP_DIR=$(expand_path "$SETUP_DIR")
  log "Running optional project setup: $SETUP_DIR"
  "$BIN_DIR/sima" setup --path "$SETUP_DIR" --backend "$BACKEND"
else
  log "Optional project setup: $BIN_DIR/sima setup --path /path/to/project"
fi

log "Done."
log "Try: $BIN_DIR/sima version"
