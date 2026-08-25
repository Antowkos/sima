#!/bin/sh
set -eu

usage() {
  cat <<'USAGE'
Usage: ./install.sh [options]

Build and install the sima CLI from this source checkout.

Options:
  --bin-dir DIR       Install directory for the sima binary (default: $HOME/.local/bin)
  --project DIR       Also initialize SIMA in a project and install managed agent instructions
  --backend auto      Add the first available backend: claude, then codex (default with --project)
  --backend claude    Add a Claude Code backend if `claude` is available
  --backend codex     Add a Codex backend if `codex` is available
  --backend none      Do not add an agent backend
  --help              Show this help

Examples:
  ./install.sh
  ./install.sh --project /path/to/project
  ./install.sh --bin-dir ~/bin --project . --backend claude
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
PROJECT_DIR=""
BACKEND="auto"
BACKEND_EXPLICIT="false"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --bin-dir)
      [ "$#" -ge 2 ] || fail "--bin-dir requires a value"
      BIN_DIR="$2"
      shift 2
      ;;
    --project)
      [ "$#" -ge 2 ] || fail "--project requires a value"
      PROJECT_DIR="$2"
      shift 2
      ;;
    --backend)
      [ "$#" -ge 2 ] || fail "--backend requires a value"
      BACKEND="$2"
      BACKEND_EXPLICIT="true"
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

if [ -n "$PROJECT_DIR" ]; then
  PROJECT_DIR=$(expand_path "$PROJECT_DIR")
  mkdir -p "$PROJECT_DIR"
  log "Initializing SIMA project state in: $PROJECT_DIR"
  "$BIN_DIR/sima" init "$PROJECT_DIR"
  "$BIN_DIR/sima" install --path "$PROJECT_DIR"

  if [ "$BACKEND" = "auto" ]; then
    if command -v claude >/dev/null 2>&1; then
      BACKEND="claude"
    elif command -v codex >/dev/null 2>&1; then
      BACKEND="codex"
    else
      BACKEND="none"
    fi
  fi

  case "$BACKEND" in
    claude)
      if command -v claude >/dev/null 2>&1; then
        "$BIN_DIR/sima" backend add claude-main --kind claude-code --executable "$(command -v claude)" --path "$PROJECT_DIR" --force
        log "Added backend: claude-main"
      elif [ "$BACKEND_EXPLICIT" = "true" ]; then
        fail "--backend claude requested but claude was not found in PATH"
      else
        log "Skipped Claude backend: claude not found in PATH"
      fi
      ;;
    codex)
      if command -v codex >/dev/null 2>&1; then
        "$BIN_DIR/sima" backend add codex-main --kind codex --executable "$(command -v codex)" --path "$PROJECT_DIR" --force
        log "Added backend: codex-main"
        log "Before the first Codex learn run, check auth with: codex doctor"
      elif [ "$BACKEND_EXPLICIT" = "true" ]; then
        fail "--backend codex requested but codex was not found in PATH"
      else
        log "Skipped Codex backend: codex not found in PATH"
      fi
      ;;
    none)
      log "Skipped backend setup. Add one later with: sima backend add ..."
      ;;
  esac

  if [ "$BACKEND" = "none" ]; then
    log "Running lint preflight..."
    "$BIN_DIR/sima" lint "$PROJECT_DIR"
    log "Skipped sima doctor because no backend is configured yet. Run it after sima backend add."
  else
    log "Running preflight..."
    "$BIN_DIR/sima" doctor "$PROJECT_DIR"
    "$BIN_DIR/sima" lint "$PROJECT_DIR"
  fi
fi

log "Done."
log "Try: $BIN_DIR/sima version"
if [ -n "$PROJECT_DIR" ]; then
  log "Next: $BIN_DIR/sima brief \"small real task\" --path $PROJECT_DIR"
fi
