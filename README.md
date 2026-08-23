# SIMA

Self Improvement Memory Agent.

SIMA is a private Go CLI for running Claude Code/Codex through a personal self-improvement loop: project-local memory, skills, evidence, clean-session archivist checks, and safe auto-application of local improvements.

## Current slice

```bash
sima init [path]
sima doctor [path]
sima brief "task description" [--path path]
sima run --backend <name> --task "task description" [--path path]
sima propose --from-run <run-id|last|path> [--path path]
sima review [--path path] [--all]
sima apply <proposal-id|path> [--path path]
sima archivist --proposal <proposal-id|path> [--path path]
sima backend list [path]
sima backend add <name> --kind <claude-code|codex> --executable <path>
sima backend doctor <name> [path]
sima version
```
