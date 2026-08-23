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
- compact snippets from active personal memory and skills, with source pointers;
- compact snippets from active team/shared scaffold entries when present;
- SDD artifact paths under `docs/specs`, `docs/plans`, and `openspec/changes`;
- safety policy reminders for reward-hacking prevention and clean-session archivist review.

Brief content is bounded: active memory/skill snippets are truncated and limited by item count so briefings stay token-sparse while still surfacing learned knowledge. Briefs are input artifacts for future `sima run` and should be preserved in run evidence.

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

`sima run --backend <name> --task <task> [--no-propose]` creates a bounded worker run under `.sima/personal/runs/<run-id>/`.

By default, a successful run immediately calls the same proposal generator as `sima propose --from-run <run-id>` and prints next-step commands for review and archivist decision. `--no-propose` disables this auto-chain for diagnostic/manual runs.

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

## One-command learn loop

`sima learn --backend <name> --task <task> [--path <path>]` runs the gated personal self-improvement loop:

```text
run -> propose -> archivist -> apply
```

It stops without applying when the worker run fails, proposal generation fails, safety/review gates reject the proposal, or the archivist returns `reject`/`defer`. Only an `archivist_decision: apply` proposal reaches `sima apply`.

Backend command mapping for v0:

- `claude-code`: `<executable> -p <prompt>`
- `codex`: `<executable> exec <prompt>`

## Proposal generation

`sima propose --from-run <run-id|last|path>` creates a personal candidate proposal under `.sima/personal/memory/candidates/` from a bounded run bundle.

The v0 proposal is intentionally conservative:

- it references task, brief, command, stdout, stderr, and `worker-report.yaml` as evidence;
- it reads structured `proposed_memory` and `proposed_skills` from `worker-report.yaml` or YAML stdout when present;
- it fills missing candidate evidence from the run artifact bundle;
- it performs deterministic safety flagging for obvious reward-hacking/test-weakening language;
- it sets `archivist_decision: defer` so a clean archivist session must review before apply;
- it emits a fallback review candidate only for successful, safe runs with no structured worker proposals.

## Proposal review

`sima review [--path <path>] [--all]` reads personal candidate proposals, validates their structure, and prints a compact review queue summary.

The v0 review gate marks an item blocked when:

- required fields are missing;
- proposal operation/status/decision/safety values are unsupported;
- evidence pointers are missing or malformed;
- candidate memories/skills lack triggerable summaries or evidence;
- a suspicious/unsafe proposal asks to `apply`.

Without `--all`, review only shows proposals with `status: candidate`.

## Proposal apply

`sima apply <proposal-id|path> [--path <path>]` promotes an approved personal proposal into active `.sima` knowledge.

v0 gates are intentionally strict:

- proposal must pass `sima review` validation;
- `status` must be `candidate`;
- `scope` must be `personal`;
- `safety.decision` must be `safe`;
- `archivist_decision` must be `apply`;
- proposal must contain at least one candidate memory or skill.

When gates pass, SIMA writes candidate memories to `.sima/personal/memory/cards/*.yaml`, candidate skills to `.sima/personal/skills/active/*.md`, and marks the source proposal `status: applied` with `applied_at`.

## Archivist decision

`sima archivist --proposal <proposal-id|path> [--path <path>]` is the deterministic v0 clean-checker gate. It updates `archivist_decision`, `archivist_at`, and `archivist_notes` on the source proposal.

Decision rules:

- `apply`: proposal is valid, `status: candidate`, `scope: personal`, `safety.decision: safe`, has at least one candidate memory/skill, and no active output file conflict exists.
- `reject`: proposal is invalid, suspicious/unsafe, or has no candidates.
- `defer`: proposal is outside v0 auto-approval scope or needs manual dedup/update because an active output already exists.

`apply` still requires a separate `sima apply` invocation so decision and mutation stay distinct.

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
