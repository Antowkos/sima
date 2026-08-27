# SIMA 5-Minute Setup

Use this guide to run the first team-alpha SIMA loop in an existing project. The goal is to verify the autonomous personal-learning path, not to configure team/shared memory yet.

## 0. Prerequisites

- Go is installed.
- At least one agent CLI is installed, usually Claude Code or Codex.
- You have a project directory where `.sima/` can be created.

Examples below use `$PROJECT` for the pilot repository and `$SIMA` for the SIMA source checkout.

## 1. Install the binary

From the SIMA source checkout:

```bash
cd $SIMA
./install.sh
```

This builds the `sima` binary and installs it to `~/.local/bin` by default. It does not mutate a pilot project unless you explicitly ask for setup.

Use a custom binary directory if needed:

```bash
./install.sh --bin-dir ~/bin
```

If the install directory is not on `PATH`, either add it or run SIMA via the printed binary path, for example `~/.local/bin/sima`.

## 2. Set up a pilot project

```bash
cd $PROJECT
sima setup
```

This initializes project-local `.sima/` state in the current directory, upserts managed `CLAUDE.md`/`AGENTS.md` instructions, auto-adds the first available Claude/Codex backend, and runs preflight checks. If you prefer not to `cd`, pass an explicit path: `sima setup --path $PROJECT`.

If you want to initialize the project without adding a backend yet:

```bash
sima setup --backend none
```

If you use a wrapper or alternate agent executable, pass it during setup:

```bash
sima setup --backend claude --executable /path/to/claude-wrapper
sima setup --backend codex --executable /path/to/codex-wrapper
sima setup --claude-executable /path/to/claude --codex-executable /path/to/codex
```

If one Claude Code executable uses multiple account/config directories, pass the config directory directly instead of relying on a shell alias:

```bash
sima setup --backend claude --claude-config-dir ~/.claude-work
# equivalent low-level form:
sima setup --backend claude --env CLAUDE_CONFIG_DIR=$HOME/.claude-work
```

Shell aliases such as `alias claude-work='CLAUDE_CONFIG_DIR=~/.claude-work claude'` are convenient interactively, but SIMA runs backends with `exec`, so aliases are not expanded. Use `--claude-config-dir`, `--env`, or a real wrapper script path with `--executable`.

For one-command onboarding from the SIMA checkout, installation can optionally call setup after the binary is installed:

```bash
./install.sh --setup $PROJECT
```

If you invoke the installer from inside the pilot repo, for example `$SIMA/install.sh --setup`, setup uses the current directory.

`./install.sh --project $PROJECT` is kept as an alias for `--setup` for existing scripts.

## 3. Manual build path

```bash
cd $SIMA
go build -o sima ./cmd/sima
```

Optional convenience copy:

```bash
mkdir -p ~/bin
cp ./sima ~/bin/sima
```

If you do not copy it into `PATH`, run SIMA via the built binary path, for example `$SIMA/sima`.

## 4. Manual project initialization

```bash
cd $PROJECT
sima init .
sima install --path .
```

This creates project-local state under `.sima/` only. The SIMA source code is not vendored into the project. `sima install` upserts managed SIMA blocks into `CLAUDE.md` and `AGENTS.md` so Claude Code and Codex see the same project-memory rules.

## 5. Manual backend add

Choose one installed executable.

Claude Code example:

```bash
sima backend add claude-main --kind claude-code --executable "$(command -v claude)" --path .
```

Codex example:

```bash
codex doctor
# If auth fails, run: codex login
sima backend add codex-main --kind codex --executable "$(command -v codex)" --path .
```

`codex doctor` should report valid auth before `sima learn`; otherwise the first real Codex run will fail with `401 Unauthorized`.

If you use a wrapper script, pass the wrapper path as `--executable`.

## 6. Run preflight

```bash
sima doctor .
sima lint .
```

Expected healthy state:

```text
[ok] config: learn auto_apply: auto_apply=true auto_cleanup_deferred=true
[ok] config: learn auto_cleanup_deferred: auto_apply=true auto_cleanup_deferred=true
[ok] backends: configured: ...
[ok] lint: errors: 0 errors, 0 warnings
[ok] candidates: queue: 0 pending candidates
```

If `sima doctor` reports no backend, run `sima backend add ...` first. If lint reports errors, fix them before running `learn`.

## 7. Create a brief for a real small task

Pick a small, real task in the pilot project. Avoid secrets, credentials, and large destructive changes.

```bash
sima brief "small real task" --path .
```

The brief should contain only active memory/skills and bounded source context.

## 8. Run auto-learning

```bash
sima learn --backend <backend-name> --task "small real task" --path .
```

Default behavior is auto-learning-first:

```text
worker → proposal → clean archivist → apply-ready gate → auto-apply
```

A successful run ends with a compact `Learn summary:` block. For wrappers or automation, use JSON output:

```bash
sima learn --backend <backend-name> --task "small real task" --json --path .
```

## 9. Inspect what changed

```bash
sima candidates list --status all --path .
sima memory list --status active --path .
sima skill list --status active --path .
sima lint .
```

Useful result:

- at least one durable personal memory or skill was auto-applied;
- `sima lint .` remains clean;
- a later `sima brief "follow-up task" --path .` includes the useful learned item.

## 10. Inspect-only escape hatch

For sensitive repos or first-run demos, disable auto-apply for one run:

```bash
sima learn --backend <backend-name> --task "small real task" --no-auto-apply --path .
sima candidates list --path .
sima candidates show <candidate-id> --path .
sima candidates apply-ready --path .
```

Apply later only if the proposal is acceptable:

```bash
sima candidates apply-ready --apply --path .
```

## What feedback to capture

For each teammate pilot, capture:

1. setup friction;
2. backend executable/config friction;
3. brief usefulness/noise;
4. proposal quality;
5. archivist decision quality;
6. whether the auto-applied memory/skill felt safe and useful;
7. whether candidates, runs, evidence, and lint made debugging clear.
