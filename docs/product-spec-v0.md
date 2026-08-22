# SIMA Product Spec v0

SIMA means Self Improvement Memory Agent.

## Goal

Build a private, single-binary CLI that lets one person/agent run Claude Code or Codex through a self-improvement loop:

1. prepare a compact brief;
2. run a bounded worker task;
3. capture artifacts;
4. draft memory and skill changes;
5. run a clean-session archivist/checker;
6. auto-apply safe personal/local improvements.

## Non-goals for v0

- Team/shared promotion workflow.
- Cursor support.
- Hosted service or web UI.
- Embeddings/vector DB.
- Human accept/edit as the default bottleneck.

## Principles

- Personal/local self-improvement first.
- Team/shared scaffold exists, but team changes are not auto-applied.
- Archivist is separate from worker and runs in a clean session.
- Reward hacking must not become learned behavior.
- User may explicitly request memory/skill updates, but they still pass checks.
