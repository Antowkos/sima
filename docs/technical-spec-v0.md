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
