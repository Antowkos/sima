# SIMA Team Alpha Readiness

SIMA is moving from solo dogfood toward a small internal team alpha. The goal is to collect feedback from real teammates while proving the core product promise: agents should safely learn from real work automatically, not just produce inspectable suggestions.

## Readiness target

Team alpha means:

- technical teammates can install or build `sima` locally;
- each pilot repo can run `sima init`, `sima doctor`, `sima lint`, and `sima brief` without manual explanation;
- at least one real Claude Code or Codex backend can pass `sima backend doctor`;
- `sima learn` runs in self-improving mode by default for personal/local learning, with inspect-only available as an override;
- all memory/skill changes remain auditable via lifecycle status and candidate history;
- team feedback is captured as issues or docs changes, not informal chat-only notes.

Team alpha does **not** mean public beta. Breaking changes are acceptable, but they must be explicit and documented.

## Auto-learning defaults for team usage

For first team pilots, keep the main path auto-learning-first while preserving explicit escape hatches:

```yaml
learn:
  auto_apply: true
  auto_cleanup_deferred: true
team:
  auto_apply: false
```

Rationale:

- auto-learning is the core SIMA value and must be exercised during team alpha, not postponed;
- personal/local memory should auto-apply when it passes archivist and deterministic apply-ready gates;
- team/shared memory should stay review-required until the workflow is trusted;
- inspect-only `sima learn --no-auto-apply` remains available for sensitive repos, first-run demonstrations, and debugging;
- cleanup can run automatically because it preserves audit history by changing deferred candidates to `status: deferred` rather than deleting them.

## Pilot workflow

Recommended first-session workflow for a teammate:

```bash
sima init .
sima doctor .
sima backend add <name> --kind <claude-code|codex> --executable <path>
sima backend doctor <name> .
sima brief "small real task" --path .
sima learn --backend <name> --task "small real task" --auto-cleanup-deferred --path .
sima candidates list --status all --path .
sima memory list --status active --path .
sima skill list --status active --path .
sima lint .
```

For an inspect-only rehearsal or sensitive repo, use:

```bash
sima learn --backend <name> --task "small real task" --no-auto-apply --auto-cleanup-deferred --path .
sima candidates show <id> --path .
sima candidates apply-ready --path .
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
6. **Auto-learning trust** — did the auto-applied personal memory/skill feel safe and useful? What would make it trustworthy enough for regular use?
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
- [ ] real Claude Code auto-learning dogfood completed on a small task;
- [ ] real Codex auto-learning dogfood completed on a small task.

## Team memory policy for alpha

Team/shared knowledge is higher-impact than personal knowledge. During alpha:

- personal proposals auto-apply by default when they pass gates;
- team proposals must remain review-required;
- team candidates should include enough evidence for another teammate to audit;
- stale or disputed team knowledge should be deprecated/superseded, not edited silently;
- `sima brief` should prefer team active knowledge over personal active knowledge when both match, but this priority can be refined after pilots.

## Success criteria

The team alpha is successful when:

- at least 2 teammates run SIMA on real tasks;
- each teammate can inspect what SIMA proposed and why;
- at least 5 concrete feedback items are captured;
- at least 1 useful memory or skill is auto-applied by `sima learn` and improves a later brief;
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
