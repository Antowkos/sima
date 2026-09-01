# SIMA Commands

This page is the compact command reference. Start with [5-Minute Setup](5-minute-setup.md) if you are installing SIMA for the first time.

## Project setup

```bash
sima version
sima init [path]
sima install [--client claude|codex|all] [--path path]
sima setup [path] [--path path] [--backend auto|claude|codex|none] [--executable path] [--claude-config-dir path] [--env KEY=VALUE]
sima doctor [path]
sima lint [path]
```

- `init` creates project-local `.sima/` storage.
- `install` writes managed `CLAUDE.md` / `AGENTS.md` instructions, Claude Code project slash commands under `.claude/commands/`, and Codex project skills under `.codex/skills/`.
- `setup` runs `init`, `install`, optional backend setup, and preflight checks.
- `doctor` checks scaffold/config/backends/lint/candidate queue health.
- `lint` checks malformed knowledge files, lifecycle status, candidate queues, and unsafe paths.

## Briefing and work execution

```bash
sima brief "task description" [--path path]
sima index rebuild [--path path]
sima run --backend <name> --task "task description" [--path path] [--no-propose]
sima learn --backend <name> --task "task description" [--archivist-backend name] [--auto-apply|--no-auto-apply] [--auto-cleanup-deferred|--no-auto-cleanup-deferred] [--json] [--path path]
```

- `brief` emits a compact sourced context packet from active memory and skills. By default it filters active knowledge with a deterministic lexical task-relevance heuristic. Projects can opt into embedding retrieval by setting `brief.retrieval: embedding` or `hybrid` and providing `brief.embedding.command`; SIMA invokes that external command per brief and does not keep the model resident unless the configured command talks to a daemon.
- `run` executes a backend with the SIMA briefing and captures evidence.
- `learn` runs the worker, parses structured learning proposals, runs clean-session archivist review, and auto-applies safe personal/local knowledge when configured.

Optional embedding retriever contract:

```yaml
brief:
  retrieval: hybrid # lexical | embedding | hybrid
  max_selected: 8
  embedding:
    command: ./scripts/sima-embed-e5.py
    model: intfloat/multilingual-e5-small
    min_score: 0.85
  query:
    decomposition: command # none | heuristic | command
    command: ./scripts/sima-split-query.py
    max_parts: 4
```

When embedding retrieval is enabled, SIMA maintains a persistent JSONL index at `.sima/index/embeddings.jsonl`. `sima apply` updates index rows for newly created, updated, superseded, or deprecated active knowledge. `sima brief` also refreshes stale/missing rows by comparing each active card/skill's metadata hash, so manual edits are picked up lazily. The default/recommended `min_score` is `0.85` for the E5 helper because unrelated text pairs often still have a high cosine baseline; tune it per project if briefs are too broad or too sparse. Existing configs are preserved during upgrades, so projects that still have `min_score: 0.2` should update that setting manually if they want filtering rather than broad reranking.

`brief.query.decomposition` can improve multi-topic tasks whose single embedding gets diluted across intents. `none` embeds the original task only. `heuristic` adds conservative local splits on conjunctions/punctuation. `command` calls an external splitter, which can be backed by a local or hosted model, and unions matches from the original task plus returned sub-queries. Splitter failures fall back to the original task only.

The query decomposition command receives JSON on stdin and returns sub-queries on stdout:

```json
{"task":"исправить guard-else-return и открыть PR","max_parts":4}
```

```json
{"queries":["исправить guard-else-return","открыть PR"]}
```

```bash
sima index rebuild --path .
```

The embedding command receives JSON on stdin and returns embeddings on stdout:

```json
{"model":"intfloat/multilingual-e5-small","texts":[{"id":"__task__","text":"task"},{"id":".sima/personal/memory/cards/x.yaml","path":".sima/personal/memory/cards/x.yaml","text":"title trigger summary"}]}
```

```json
{"embeddings":[{"id":"__task__","vector":[0.1,0.2]},{"id":".sima/personal/memory/cards/x.yaml","vector":[0.1,0.2]}]}
```

## Explicit memory

```bash
sima remember "durable project knowledge" \
  [--source user|review|agent] \
  [--type decision|invariant|gotcha|workflow|guardrail|anti_pattern|open_question] \
  [--title title] \
  [--trigger "When ..."] \
  [--backend name] \
  [--path path]
```

Use `remember` when a user explicitly asks an agent to save project knowledge. Managed Claude/Codex instructions route “remember/save/learn/record this” requests through SIMA rather than native/simple memory.

## Candidates, review, and apply

```bash
sima propose --from-run <run-id|last|path> [--path path]
sima review [--path path] [--all]
sima candidates list [--status candidate|deferred|applied|rejected|all] [--path path]
sima candidates show <id|path> [--path path]
sima candidates apply-ready [--apply] [--path path]
sima candidates cleanup [--path path]
sima apply <proposal-id|path> [--path path]
sima archivist --proposal <proposal-id|path> [--backend name] [--path path]
```

- `candidates apply-ready` filters proposals through deterministic apply gates.
- `--apply` mutates only apply-ready proposals.
- `cleanup` marks deferred proposals as deferred without deleting audit history.
- `archivist` runs a clean model-backed reviewer over bounded evidence and active knowledge context.

## Audit lists

```bash
sima memory list [--status active|deprecated|superseded|archived|all] [--path path]
sima skill list [--status active|deprecated|superseded|archived|all] [--path path]
```

Only explicit `status: active` knowledge is injected into future briefs. Deprecated/superseded/archived items stay auditable but do not enter active context.

## Backend profiles

```bash
sima backend list [path]
sima backend add <name> --kind <claude-code|codex> --executable <path> [--permission-mode workspace-write] [--path path]
sima backend doctor <name> [path]
```

Backends are named profiles. They can point to different Claude/Codex binaries, config dirs, env files, wrappers, working directories, permission modes, or metadata.

## Team flow

Consumption is implemented first; promotion stays review-required and will be added separately.

```bash
sima team init --repo <git-url> [--ref main] [--path path]
sima team pull [--path path]
sima team status [--path path]
```

- `team init` stores the shared knowledge repo in `.sima/config.yaml` with `auto_apply: false` and `sync_mode: mirror`.
- `team pull` clones/fetches the repo into `.sima/team/source` and mirrors reviewed files into `.sima/team/memory/cards` and `.sima/team/skills/active`.
- `team status` reports configuration, clone status, and local mirror counts.
- Expected knowledge repo layout:

```text
memory/cards/*.yaml
skills/active/*.md
```

Intended model:

```text
local learn → personal active → explicit team propose PR → review/merge → team pull → brief
```

Planned next command:

```bash
sima team propose <memory-or-skill-id|path> --path .
```
