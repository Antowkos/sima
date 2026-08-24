# SIMA

Self Improvement Memory Agent.

SIMA is a private Go CLI for running Claude Code/Codex through a personal self-improvement loop: project-local memory, skills, evidence, clean-session archivist checks, and safe auto-application of local improvements.

Generated briefs include bounded snippets from active memory cards and skills, so applied lessons feed back into later runs without dumping raw artifacts into context. `sima propose` can turn structured worker JSON `proposed_memory` / `proposed_skills` output into reviewable candidates. Fallback review candidates stay deferred; `sima learn` auto-applies only safe structured worker proposals. Malformed or incomplete structured output is marked `candidate_source: structured_invalid` with `candidate_errors`, never silently converted to fallback.

For Claude Code backends, set `metadata.output_format: json_schema` to run with Claude's native `--output-format json --json-schema` mode; SIMA extracts the validated `structured_output` wrapper field.

Review gates follow Hermes-style learning hygiene: durable memory must be compact, triggerable, evidence-backed, and not transient task progress; skills must describe reusable workflows rather than one-off run summaries. Proposals persist a small librarian classification (`destination`, `operation`, `quality`) so review can distinguish memory, skill, mixed, session-only, and rejected learning paths.

## Current slice

```bash
sima init [path]
sima doctor [path]
sima brief "task description" [--path path]
sima run --backend <name> --task "task description" [--path path] [--no-propose]
sima learn --backend <name> --task "task description" [--path path]
sima propose --from-run <run-id|last|path> [--path path]
sima review [--path path] [--all]
sima apply <proposal-id|path> [--path path]
sima archivist --proposal <proposal-id|path> [--path path]
sima backend list [path]
sima backend add <name> --kind <claude-code|codex> --executable <path>
sima backend doctor <name> [path]
sima version
```
