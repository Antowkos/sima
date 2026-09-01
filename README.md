# SIMA

**Self Improvement Memory Agent**

SIMA is a project-local memory and skill layer for coding agents such as Claude Code and Codex.

The core mental model:

```text
work → evidence → local learning → clean archivist review → active memory/skill → better future brief
```

SIMA is not a replacement for normal coding workflows. Agents still inspect repos, use `gh`, edit files, run tests, and verify work normally. SIMA adds a durable learning loop around that work so useful project knowledge survives across fresh agent sessions without dumping raw logs or chat history into context.

## Install

### From the latest GitHub Release

Use GitHub CLI access:

```bash
mkdir -p /tmp/sima-install
cd /tmp/sima-install

gh release download v0.1.0-alpha.2 \
  --repo Antowkos/sima \
  --pattern 'sima_0.1.0-alpha.2_darwin_arm64.tar.gz'

tar -xzf sima_0.1.0-alpha.2_darwin_arm64.tar.gz
install -m 0755 sima_0.1.0-alpha.2_darwin_arm64/sima ~/.local/bin/sima

sima version
```

Features currently on `main` but not yet tagged are available by building from source until the next release.

Pick the matching archive for your platform:

- `darwin_arm64`
- `darwin_amd64`
- `linux_arm64`
- `linux_amd64`
- `windows_amd64`

### Build from source

From a SIMA source checkout:

```bash
./install.sh
sima version
```

Then initialize a project explicitly:

```bash
cd /path/to/project
sima setup
sima doctor .
sima lint .
```

For install details, repo-URL agent bootstrap, release process, and agent-specific usage, see:

- [5-Minute Setup](docs/5-minute-setup.md)
- [Agent Bootstrap](docs/agent-bootstrap.md)
- [Using SIMA with Agents](docs/agent-usage.md)
- [Release Process](docs/release-process.md)

## Use inside agents

SIMA is designed to be used by agents during real work, not only by humans from a terminal.

After setup:

```bash
cd /path/to/project
sima setup
```

SIMA writes:

```text
.sima/                         # project-local memory, skills, runs, evidence, config
CLAUDE.md                      # Claude Code project instructions
AGENTS.md                      # Codex/OpenAI agent project instructions
.claude/commands/sima.md       # Claude slash command: /sima <task>
.claude/commands/sima-brief.md # Claude slash command: /sima-brief <task>
.claude/commands/sima-remember.md # Claude slash command: /sima-remember <knowledge>
.codex/skills/sima/SKILL.md    # Codex skill for /sima-like SIMA flow
.codex/skills/sima-brief/SKILL.md # Codex skill for SIMA briefing
.codex/skills/sima-remember/SKILL.md # Codex skill for durable memory requests
```

`CLAUDE.md` and `AGENTS.md` receive an upserted managed block:

```html
<!-- BEGIN SIMA MANAGED INSTRUCTIONS -->
...
<!-- END SIMA MANAGED INSTRUCTIONS -->
```

Content outside the managed block is preserved. Re-running `sima setup` or `sima install` updates the SIMA block without replacing the rest of the file.

### How to prompt Claude/Codex

Use a prompt like:

```text
Use SIMA for this task. Start with `sima brief`, do the normal repo workflow, verify with tests, then run `sima learn` only for durable lessons.

Task: <your task>
```

Typical agent commands:

```bash
sima brief "<task>" --path .
# inspect repo, edit files, run tests/builds normally
sima learn --backend <backend-name> --task "<task>" --path .
```

If embedding retrieval is enabled in `.sima/config.yaml`, rebuild vectors for existing or bulk-edited knowledge with:

```bash
sima index rebuild --path .
```

SIMA stores those vectors in `.sima/index/embeddings.jsonl`. New/updated learned cards refresh the index during `sima apply`; manual edits are detected by metadata hashes and refreshed lazily during `sima brief`. For E5-style embeddings the default `min_score` is `0.85`; raise/lower it per project if briefs are too broad or too sparse.

For explicit memory requests, agents should call:

```bash
sima remember "<durable project knowledge>" \
  --source user \
  --type <decision|invariant|gotcha|workflow|guardrail|anti_pattern|open_question> \
  --trigger "When ..." \
  --path .
```

Claude Code reads `CLAUDE.md`; Codex/OpenAI agents read `AGENTS.md`. Claude Code also gets project slash commands, so you can type:

```text
/sima fix the failing auth tests
/sima-brief plan the database migration
/sima-remember API handlers must use generated request types
```

See [Using SIMA with Agents](docs/agent-usage.md) for exact Claude/Codex flows, PR-review usage, backend setup, and safety rules. Codex gets project skills under `.codex/skills/...`; verified Codex prompt input includes these skills, so `/sima`-style prompts can route through the SIMA skill path when Codex passes them to the model.

## Core flow

```text
1. User asks for work
2. Agent/SIMA runs `sima brief`
3. Agent does normal repo/tool workflow
4. Agent verifies with tests/builds/checks
5. SIMA stores run evidence
6. Worker proposes only durable memory/skills, if any
7. Clean archivist reviews the proposal
8. Deterministic apply gates check safety, lifecycle, conflicts, and schema
9. Safe personal/local knowledge becomes active
10. Later briefs retrieve active knowledge only
```

Important rules:

- personal/local learning can auto-apply after archivist + apply-ready gates;
- transient task progress should stay in run evidence, not active memory;
- malformed structured output becomes inspectable candidate data, not silent memory;
- inactive/deprecated/superseded/archived knowledge stays auditable but is not injected into briefs;
- skills are for reusable workflows, not one-off summaries.

## Team flow

Team knowledge should be promoted, not born shared.

The intended team model is:

```text
local learn → personal active memory/skill → explicit team proposal → PR review → merge → team pull → future briefs
```

A developer or agent first proves a memory/skill locally. If it is useful beyond one person, SIMA should propose it to a shared team knowledge repository through a normal pull request. Team knowledge becomes authoritative only after review and merge.

Implemented consumption commands:

```bash
sima team init --repo <git-url> [--ref main] --path .
sima team pull --path .
sima team status --path .
```

`team init` stores a read-only mirror config:

```yaml
team:
  repo: <git-url>
  ref: main
  auto_apply: false
  sync_mode: mirror
```

`team pull` clones/fetches the shared knowledge repo into `.sima/team/source` and mirrors these reviewed artifacts into the local project:

```text
memory/cards/*   → .sima/team/memory/cards/*
skills/active/*  → .sima/team/skills/active/*
```

`team status` reports whether the repo is configured/cloned and how many team memory cards/skills are available locally.

Planned promotion command: `team propose` should create a reviewable PR containing the memory/skill, trigger, evidence, source local item, safety notes, and rationale for why it belongs in team scope.

## Command reference

See [Commands](docs/commands.md) for the detailed CLI reference.

Common commands:

```bash
sima setup
sima doctor .
sima lint .
sima brief "task" --path .
sima index rebuild --path .
sima learn --backend <name> --task "task" --path .
sima remember "knowledge" --source user --type invariant --trigger "When ..." --path .
sima candidates list --status all --path .
sima memory list --status active --path .
sima skill list --status active --path .
sima backend list .
```

## License

MIT License. See [LICENSE](LICENSE).

## Author

Created by **Anton Kovalev**.
