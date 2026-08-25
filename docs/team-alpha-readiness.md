# SIMA Team Alpha Readiness

SIMA is moving from solo dogfood toward a small internal team alpha. The goal is to collect feedback from real teammates while keeping the tool safe, inspectable, and easy to recover from.

## Readiness target

Team alpha means:

- technical teammates can install or build `sima` locally;
- each pilot repo can run `sima init`, `sima doctor`, `sima lint`, and `sima brief` without manual explanation;
- at least one real Claude Code or Codex backend can pass `sima backend doctor`;
- `sima learn` can run in either self-improving mode or inspect-only mode;
- all memory/skill changes remain auditable via lifecycle status and candidate history;
- team feedback is captured as issues or docs changes, not informal chat-only notes.

Team alpha does **not** mean public beta. Breaking changes are acceptable, but they must be explicit and documented.

## Safety defaults for team usage

For first team pilots, prefer conservative defaults:

```yaml
learn:
  auto_apply: false
  auto_cleanup_deferred: true
team:
  auto_apply: false
```

Rationale:

- personal/local memory may eventually auto-apply by default;
- team/shared memory should stay review-required until the workflow is trusted;
- inspect-only `sima learn --no-auto-apply` is the safest default for teammates learning the tool;
- cleanup can run automatically because it preserves audit history by changing deferred candidates to `status: deferred` rather than deleting them.

## Pilot workflow

Recommended first-session workflow for a teammate:

```bash
sima init .
sima doctor .
sima backend add <name> --kind <claude-code|codex> --executable <path>
sima backend doctor <name> .
sima brief "small real task" --path .
sima learn --backend <name> --task "small real task" --no-auto-apply --auto-cleanup-deferred --path .
sima candidates list --status all --path .
sima candidates show <id> --path .
sima candidates apply-ready --path .
sima lint .
```

Only after inspecting the candidate:

```bash
sima candidates apply-ready --apply --path .
sima brief "follow-up task" --path .
```

## Feedback to collect

For each pilot run, collect:

1. **Setup friction** — what command/config step was unclear?
2. **Backend friction** — did Claude/Codex return valid JSON/schema output?
3. **Brief quality** — was retrieved memory/skill context useful or noisy?
4. **Proposal quality** — did proposed memory/skill feel durable, triggerable, and non-transient?
5. **Archivist quality** — did apply/defer/reject match human judgment?
6. **Apply trust** — would the teammate trust auto-apply for personal memory? For team memory?
7. **Recovery/debuggability** — could they inspect runs, candidates, evidence, and lint results?
8. **Missing integrations** — Claude commands, Codex `AGENTS.md`, CI, GitHub issues, Slack/Telegram reporting, etc.

## Alpha blockers

Before inviting more than 1–2 technical teammates, complete:

- [ ] config-driven learn defaults in `.sima/config.yaml`;
- [ ] `sima doctor` covers config, directories, lint, and candidate queue health;
- [ ] `sima backend doctor` verifies executable, prompt round-trip, JSON output, and schema mode when configured;
- [ ] `sima learn` prints a concise final summary;
- [ ] `sima learn --json` or equivalent machine-readable summary exists for wrappers;
- [ ] short 5-minute setup guide exists;
- [ ] real Claude Code dogfood completed on a small task;
- [ ] real Codex dogfood completed on a small task.

## Team memory policy for alpha

Team/shared knowledge is higher-impact than personal knowledge. During alpha:

- personal proposals may be inspect-only or auto-applied depending on config;
- team proposals must remain review-required;
- team candidates should include enough evidence for another teammate to audit;
- stale or disputed team knowledge should be deprecated/superseded, not edited silently;
- `sima brief` should prefer team active knowledge over personal active knowledge when both match, but this priority can be refined after pilots.

## Success criteria

The team alpha is successful when:

- at least 2 teammates run SIMA on real tasks;
- each teammate can inspect what SIMA proposed and why;
- at least 5 concrete feedback items are captured;
- at least 1 useful memory or skill survives review and improves a later brief;
- `sima lint` remains clean after pilot runs;
- no secrets or transient task logs are promoted to active memory.

## Suggested implementation order

1. Config defaults:
   - `learn.auto_apply`;
   - `learn.auto_cleanup_deferred`;
   - CLI flags override config.
2. `sima doctor` alpha preflight.
3. `sima backend doctor` real backend preflight.
4. `sima learn` final summary and `--json`.
5. 5-minute setup guide.
6. Managed instructions for Claude Code / Codex.
7. Team-scope review-required flow.
