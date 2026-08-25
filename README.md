# SIMA

Self Improvement Memory Agent.

SIMA is a private Go CLI for running Claude Code/Codex through a personal self-improvement loop: project-local memory, skills, evidence, clean-session archivist checks, and safe auto-application of local improvements.

Generated briefs include bounded snippets from active memory cards and skills, so applied lessons feed back into later runs without dumping raw artifacts into context. `sima propose` can turn structured worker JSON `proposed_memory` / `proposed_skills` output into reviewable candidates. SIMA does not synthesize fallback learning candidates; if the worker proposes no structured learning, `sima learn` stops without archivist/apply. Malformed or incomplete structured output is marked `candidate_source: structured_invalid` with `candidate_errors`, never silently converted.

For Claude Code backends, set `metadata.output_format: json_schema` to run with Claude's native `--output-format json --json-schema` mode; SIMA extracts the validated `structured_output` wrapper field. Worker and archivist schema contracts live in shared code with explicit enums and `additionalProperties: false` so prompts, JSON Schema mode, and CLI validation do not drift.

Review gates follow Hermes-style learning hygiene: durable memory must be compact, triggerable, evidence-backed, and not transient task progress; skills must describe reusable workflows rather than one-off run summaries. Proposals persist a small librarian classification (`destination`, `operation`, `target`, `quality`) so review can distinguish memory, skill, mixed, session-only, and rejected learning paths; single-candidate collisions with active memory/skills are classified as `operation: update` with an explicit target, while ambiguous collisions are deferred for review. The target architecture is Hermes-like: deterministic code owns flow/schema/path/apply/retrieval hard gates, while model backends provide semantic candidate and archivist judgment. `sima learn` defaults to a clean model-backed archivist by reusing the worker backend for a separate reviewer invocation; `--archivist-backend` can name a distinct reviewer. `sima archivist --proposal <id> --backend <backend>` runs a clean model-backed archivist over proposal YAML, full UTF-8 evidence file contents, and active knowledge context; packets are bounded by safe project-root scope rather than byte/character truncation. CLI validation can still downgrade/reject invalid or unsafe output. `sima apply` honors lifecycle operations: `create` writes new active knowledge, `update` rewrites an explicit target, `supersede` marks a target superseded and creates replacement knowledge, and `deprecate` marks a target deprecated. `sima brief` retrieves only active knowledge and excludes deprecated/superseded/archived items.

## Current slice

```bash
sima init [path]
sima doctor [path]
sima brief "task description" [--path path]
sima run --backend <name> --task "task description" [--path path] [--no-propose]
sima learn --backend <name> --task "task description" [--archivist-backend name] [--path path]
sima propose --from-run <run-id|last|path> [--path path]
sima review [--path path] [--all]
sima apply <proposal-id|path> [--path path]
sima archivist --proposal <proposal-id|path> [--backend name] [--path path]
sima backend list [path]
sima backend add <name> --kind <claude-code|codex> --executable <path>
sima backend doctor <name> [path]
sima version
```
