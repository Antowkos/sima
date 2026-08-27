# SIMA Team Alpha Readiness

SIMA is moving from solo dogfood toward a small internal team alpha. The goal is to collect feedback from real teammates while proving the core product promise: agents should safely learn from real work automatically, not just produce inspectable suggestions.

## Readiness target

Team alpha means:

- technical teammates can install `sima` with a plain binary-only `./install.sh`, then explicitly choose whether to run `sima setup` from a pilot repo or `sima setup --path <repo>`;
- each pilot repo can run `sima setup`, `sima init`, `sima install`, `sima doctor`, `sima lint`, and `sima brief` without manual explanation;
- at least one real Claude Code or Codex backend executable is configured and visible in `sima doctor`;
- `sima learn` runs in self-improving mode by default for personal/local learning, with inspect-only available as an override;
- explicit user/review knowledge can be routed through `sima remember` rather than Claude/Codex native memory;
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
./install.sh
cd $PROJECT
sima setup
sima doctor .
sima brief "small real task" --path .
sima remember "durable project knowledge" --source user --type invariant --trigger "When this knowledge is relevant." --path .
sima learn --backend <name> --task "small real task" --path .
sima learn --backend <name> --task "small real task" --json --path .
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
2. **Backend friction** — did Claude/Codex executable/config resolution work, did Codex auth pass `codex doctor`, and did the first real `sima learn` run produce usable structured output?
3. **Brief quality** — was retrieved memory/skill context useful or noisy?
4. **Proposal quality** — did proposed memory/skill feel durable, triggerable, and non-transient?
5. **Explicit memory routing** — when asked to remember project knowledge, did Claude/Codex use `sima remember` instead of native/simple memory?
6. **Archivist quality** — did apply/defer/reject match human judgment?
7. **Auto-learning trust** — did the auto-applied personal memory/skill feel safe and useful? What would make it trustworthy enough for regular use?
8. **Recovery/debuggability** — could they inspect runs, candidates, evidence, and lint results?
9. **Missing integrations** — Claude commands, Codex `AGENTS.md`, CI, GitHub issues, Slack/Telegram reporting, etc.

## Alpha blockers

Before inviting more than 1–2 technical teammates, complete:

- [x] config-driven learn defaults in `.sima/config.yaml`;
- [x] `sima doctor` covers config, directories, auto-learning defaults, backend executables, lint, and candidate queue health;
- [x] `sima learn` prints a concise final summary;
- [x] `sima learn --json` emits a machine-readable summary for wrappers;
- [x] `sima install` writes managed Claude Code/Codex project instructions;
- [x] `sima setup` initializes a pilot repo with managed instructions plus backend/preflight setup;
- [x] `install.sh` is binary-only by default and offers setup as an explicit opt-in (`--setup` / legacy `--project`);
- [x] short [5-minute setup guide](5-minute-setup.md) exists;
- [x] real Claude Code auto-learning dogfood completed on a small task;
  - result: `claude-schema` completed worker → structured proposal → clean archivist → apply-ready → auto-apply;
  - follow-up: dogfood exposed duplicate mirrored `result`/`structured_output` candidates from Claude JSON Schema output; parser now prefers validated `structured_output` unless `result` contains an explicit lifecycle operation.
- [ ] real Codex auto-learning dogfood completed on a small task;
  - latest attempt reached real Codex CLI `0.149.1` via `npx @openai/codex`, but stopped before worker output because `codex doctor` reported missing credentials and `sima learn` failed with `401 Unauthorized`; run `codex login` or configure a supported API key before retrying.

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

## Suggested remaining implementation order

1. Real Claude Code auto-learning dogfood on a small task.
2. Real Codex auto-learning dogfood on a small task.
3. Team-scope review-required flow.
