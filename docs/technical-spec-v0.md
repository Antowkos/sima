# SIMA Technical Spec v0

## Implementation

- Language: Go.
- Distribution target: single `sima` binary.
- Repository: standalone private tool repo.
- User projects receive `.sima/` runtime state via `sima init`.

## Project-local layout

```text
.sima/
  personal/
    memory/{cards,candidates,archive}/
    skills/{active,candidates}/
    runs/
    briefs/
    evidence/
  team/
    memory/{cards,candidates,archive}/
    skills/{active,candidates}/
  system/
    skills/
    prompts/
  config.yaml
  schema.yaml
```

## Brief generation

`sima brief <task>` creates a compact task briefing under `.sima/personal/briefs/`.

The v0 brief includes:

- the task;
- system skills created by `sima init`;
- active personal memory and skills;
- active team/shared scaffold entries when present;
- SDD artifacts under `docs/specs`, `docs/plans`, and `openspec/changes`;
- safety policy reminders for reward-hacking prevention and clean-session archivist review.

Briefs are input artifacts for future `sima run` and should be preserved in run evidence.

## Backend profiles

SIMA must support multiple Claude Code/Codex installations and configs on the same machine. Backends are named profiles in `.sima/config.yaml`:

```yaml
backends:
  claude-main:
    kind: claude-code
    executable: /opt/homebrew/bin/claude
    config_path: ~/.claude/settings.json
  codex-work:
    kind: codex
    executable: ~/bin/codex-work
    config_path: ~/.codex-work/config.toml
```

Commands:

```bash
sima backend list
sima backend add <name> --kind <claude-code|codex> --executable <path>
sima backend doctor <name>
```

## SDD support

SIMA treats spec-driven-development artifacts as first-class source artifacts:

- product specs, technical specs, OpenSpec changes, and implementation plans are preserved as evidence/source material;
- `brief` should extract constraints, acceptance criteria, open questions, and verification gates;
- SIMA should not save whole specs as active memory;
- after execution, SIMA should learn only durable decisions, invariants, gotchas, guardrails, anti-patterns, or reusable skills with evidence pointers back to specs.

## Run artifact capture

`sima run --backend <name> --task <task>` creates a bounded worker run under `.sima/personal/runs/<run-id>/`.

The v0 run directory contains:

```text
task.md
brief.md
command.txt
stdout.log
stderr.log
worker-report.yaml
```

The command first generates a brief, copies it into the run directory, builds a backend-specific prompt, executes the named backend profile, and records stdout/stderr plus a structured worker report. This run artifact bundle is the input for `sima propose` and the later clean archivist/auto-apply flow.

Backend command mapping for v0:

- `claude-code`: `<executable> -p <prompt>`
- `codex`: `<executable> exec <prompt>`

## Proposal generation

`sima propose --from-run <run-id|last|path>` creates a personal candidate proposal under `.sima/personal/memory/candidates/` from a bounded run bundle.

The v0 proposal is intentionally conservative:

- it references task, brief, command, stdout, stderr, and `worker-report.yaml` as evidence;
- it performs deterministic safety flagging for obvious reward-hacking/test-weakening language;
- it sets `archivist_decision: defer` so a clean archivist session must review before apply;
- it emits a starter candidate only for successful, safe runs, and still requires evidence-backed review.

## Archivist contract

The archivist must run in a clean separate process/session from the worker. It receives bounded evidence only:

- original task;
- pre-task brief;
- diff;
- logs;
- verification results;
- worker report;
- proposed memory/skill changes;
- relevant existing memory/skills.

It emits structured decisions: `apply`, `reject`, or `defer`.

## Safety

Auto-apply only personal/local proposals with:

- valid schema;
- evidence;
- no unresolved conflict;
- clean archivist approval;
- no reward-hacking/destructive shortcut flags.
