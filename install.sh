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
  --setup [DIR]       Optionally run `sima setup` after installing (default DIR: current directory)
  --project DIR       Alias for --setup DIR, kept for existing docs/scripts
  --backend MODE      Backend mode for optional setup: auto, claude, codex, none (default: auto)
  --executable PATH   Backend executable for optional setup with --backend claude|codex
  --claude-executable PATH
                      Claude Code executable/wrapper for optional setup
  --codex-executable PATH
                      Codex executable/wrapper for optional setup
  --help              Show this help

Examples:
  ./install.sh
  ./install.sh --bin-dir ~/bin
  ./install.sh --setup
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

ORIGINAL_CWD=$(pwd)

resolve_invocation_path() {
  expanded=$(expand_path "$1")
  case "$expanded" in
    /*) printf '%s\n' "$expanded" ;;
    *) printf '%s/%s\n' "$ORIGINAL_CWD" "$expanded" ;;
  esac
}

resolve_invocation_command() {
  case "$1" in
    */*) resolve_invocation_path "$1" ;;
    *) printf '%s\n' "$1" ;;
  esac
}

BIN_DIR="${SIMA_INSTALL_DIR:-$HOME/.local/bin}"
SETUP_DIR=""
BACKEND="auto"
BACKEND_EXECUTABLE=""
CLAUDE_EXECUTABLE=""
CODEX_EXECUTABLE=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --bin-dir)
      [ "$#" -ge 2 ] || fail "--bin-dir requires a value"
      BIN_DIR="$2"
      shift 2
      ;;
    --setup)
      if [ "$#" -ge 2 ] && [ "${2#-}" = "$2" ]; then
        SETUP_DIR="$2"
        shift 2
      else
        SETUP_DIR="$ORIGINAL_CWD"
        shift 1
      fi
      ;;
    --project)
      [ "$#" -ge 2 ] || fail "--project requires a value"
      SETUP_DIR="$2"
      shift 2
      ;;
    --backend)
      [ "$#" -ge 2 ] || fail "--backend requires a value"
      BACKEND="$2"
      shift 2
      ;;
    --executable|--backend-executable)
      [ "$#" -ge 2 ] || fail "$1 requires a value"
      BACKEND_EXECUTABLE="$2"
      shift 2
      ;;
    --claude-executable)
      [ "$#" -ge 2 ] || fail "--claude-executable requires a value"
      CLAUDE_EXECUTABLE="$2"
      shift 2
      ;;
    --codex-executable)
      [ "$#" -ge 2 ] || fail "--codex-executable requires a value"
      CODEX_EXECUTABLE="$2"
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

BIN_DIR=$(resolve_invocation_path "$BIN_DIR")
if [ -n "$SETUP_DIR" ]; then
  SETUP_DIR=$(resolve_invocation_path "$SETUP_DIR")
fi
if [ -n "$BACKEND_EXECUTABLE" ]; then
  BACKEND_EXECUTABLE=$(resolve_invocation_command "$BACKEND_EXECUTABLE")
fi
if [ -n "$CLAUDE_EXECUTABLE" ]; then
  CLAUDE_EXECUTABLE=$(resolve_invocation_command "$CLAUDE_EXECUTABLE")
fi
if [ -n "$CODEX_EXECUTABLE" ]; then
  CODEX_EXECUTABLE=$(resolve_invocation_command "$CODEX_EXECUTABLE")
fi
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
  log "Running optional project setup: $SETUP_DIR"
  set -- setup --path "$SETUP_DIR" --backend "$BACKEND"
  if [ -n "$BACKEND_EXECUTABLE" ]; then
    set -- "$@" --executable "$BACKEND_EXECUTABLE"
  fi
  if [ -n "$CLAUDE_EXECUTABLE" ]; then
    set -- "$@" --claude-executable "$CLAUDE_EXECUTABLE"
  fi
  if [ -n "$CODEX_EXECUTABLE" ]; then
    set -- "$@" --codex-executable "$CODEX_EXECUTABLE"
  fi
  "$BIN_DIR/sima" "$@"
else
  log "Optional project setup: cd /path/to/project && $BIN_DIR/sima setup"
fi

log "Done."
log "Try: $BIN_DIR/sima version"
