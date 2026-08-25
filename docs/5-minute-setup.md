# SIMA 5-Minute Setup

Use this guide to run the first team-alpha SIMA loop in an existing project. The goal is to verify the autonomous personal-learning path, not to configure team/shared memory yet.

## 0. Prerequisites

- Go is installed.
- At least one agent CLI is installed, usually Claude Code or Codex.
- You have a project directory where `.sima/` can be created.

Examples below use `$PROJECT` for the pilot repository and `$SIMA` for the SIMA source checkout.

## 1. Build `sima`

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

## 2. Initialize SIMA in a pilot project

```bash
cd $PROJECT
sima init .
```

This creates project-local state under `.sima/` only. The SIMA source code is not vendored into the project.

## 3. Add one backend

Choose one installed executable.

Claude Code example:

```bash
sima backend add claude-main --kind claude-code --executable "$(command -v claude)" --path .
```

Codex example:

```bash
sima backend add codex-main --kind codex --executable "$(command -v codex)" --path .
```

If you use a wrapper script, pass the wrapper path as `--executable`.

## 4. Run preflight

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

## 5. Create a brief for a real small task

Pick a small, real task in the pilot project. Avoid secrets, credentials, and large destructive changes.

```bash
sima brief "small real task" --path .
```

The brief should contain only active memory/skills and bounded source context.

## 6. Run auto-learning

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

## 7. Inspect what changed

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

## 8. Inspect-only escape hatch

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
