# SIMA Commands

This page is the compact command reference. Start with [5-Minute Setup](5-minute-setup.md) if you are installing SIMA for the first time.

## Project setup

```bash
sima version
sima init [path]
sima install [--client claude|codex|all] [--path path]
sima setup [path] [--path path] [--backend auto|claude|codex|none] [--executable path] [--claude-config-dir path] [--env KEY=VALUE]
sima doctor [path]
sima lint [path]
```

- `init` creates project-local `.sima/` storage.
- `install` writes managed `CLAUDE.md` / `AGENTS.md` instructions, Claude Code project slash commands under `.claude/commands/`, and Codex project skills under `.codex/skills/`.
- `setup` runs `init`, `install`, optional backend setup, and preflight checks.
- `doctor` checks scaffold/config/backends/lint/candidate queue health.
- `lint` checks malformed knowledge files, lifecycle status, candidate queues, and unsafe paths.

## Briefing and work execution

```bash
sima brief "task description" [--path path]
sima run --backend <name> --task "task description" [--path path] [--no-propose]
sima learn --backend <name> --task "task description" [--archivist-backend name] [--auto-apply|--no-auto-apply] [--auto-cleanup-deferred|--no-auto-cleanup-deferred] [--json] [--path path]
```

- `brief` emits a compact sourced context packet from active memory and skills.
- `run` executes a backend with the SIMA briefing and captures evidence.
- `learn` runs the worker, parses structured learning proposals, runs clean-session archivist review, and auto-applies safe personal/local knowledge when configured.

## Explicit memory

```bash
sima remember "durable project knowledge" \
  [--source user|review|agent] \
  [--type decision|invariant|gotcha|workflow|guardrail|anti_pattern|open_question] \
  [--title title] \
  [--trigger "When ..."] \
  [--backend name] \
  [--path path]
```

Use `remember` when a user explicitly asks an agent to save project knowledge. Managed Claude/Codex instructions route “remember/save/learn/record this” requests through SIMA rather than native/simple memory.

## Candidates, review, and apply

```bash
sima propose --from-run <run-id|last|path> [--path path]
sima review [--path path] [--all]
sima candidates list [--status candidate|deferred|applied|rejected|all] [--path path]
sima candidates show <id|path> [--path path]
sima candidates apply-ready [--apply] [--path path]
sima candidates cleanup [--path path]
sima apply <proposal-id|path> [--path path]
sima archivist --proposal <proposal-id|path> [--backend name] [--path path]
```

- `candidates apply-ready` filters proposals through deterministic apply gates.
- `--apply` mutates only apply-ready proposals.
- `cleanup` marks deferred proposals as deferred without deleting audit history.
- `archivist` runs a clean model-backed reviewer over bounded evidence and active knowledge context.

## Audit lists

```bash
sima memory list [--status active|deprecated|superseded|archived|all] [--path path]
sima skill list [--status active|deprecated|superseded|archived|all] [--path path]
```

Only explicit `status: active` knowledge is injected into future briefs. Deprecated/superseded/archived items stay auditable but do not enter active context.

## Backend profiles

```bash
sima backend list [path]
sima backend add <name> --kind <claude-code|codex> --executable <path> [--permission-mode workspace-write] [--path path]
sima backend doctor <name> [path]
```

Backends are named profiles. They can point to different Claude/Codex binaries, config dirs, env files, wrappers, working directories, permission modes, or metadata.

## Team flow planned commands

The team flow is planned, not fully implemented yet:

```bash
sima team init --repo <git-url> --path .
sima team pull --path .
sima team status --path .
sima team propose <memory-or-skill-id|path> --path .
```

Intended model:

```text
local learn → personal active → explicit team propose PR → review/merge → team pull → brief
```
