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

For the current private alpha repo, use authenticated GitHub CLI access:

```bash
mkdir -p /tmp/sima-install
cd /tmp/sima-install

gh release download v0.1.0-alpha.1 \
  --repo Antowkos/sima \
  --pattern 'sima_0.1.0-alpha.1_darwin_arm64.tar.gz'

tar -xzf sima_0.1.0-alpha.1_darwin_arm64.tar.gz
install -m 0755 sima_0.1.0-alpha.1_darwin_arm64/sima ~/.local/bin/sima

sima version
```

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

For install details, repo-URL agent bootstrap, and release process, see:

- [5-Minute Setup](docs/5-minute-setup.md)
- [Agent Bootstrap](docs/agent-bootstrap.md)
- [Release Process](docs/release-process.md)

## Use as a regular user

Most users need only four commands.

### 1. Set up SIMA in a project

```bash
cd /path/to/project
sima setup
```

This creates `.sima/`, installs managed `CLAUDE.md` / `AGENTS.md` instructions, configures the first available Claude/Codex backend, and runs preflight checks.

### 2. Ask for a task briefing

```bash
sima brief "fix the failing auth tests" --path .
```

The brief contains compact snippets from active project memory and skills. It does not paste raw run logs or old conversations into context.

### 3. Let SIMA run a bounded learning task

```bash
sima learn --backend codex-main --task "fix the failing auth tests" --path .
```

SIMA runs the backend with a briefing, captures evidence, asks the worker for structured learning only if it discovered something durable, reviews the proposal in a clean archivist session, and applies safe personal/local memory or skills when gates pass.

### 4. Save explicit project knowledge

```bash
sima remember "Use the generated client for API calls; do not hand-write endpoint strings." \
  --source user \
  --type invariant \
  --trigger "When editing API client code." \
  --path .
```

Claude/Codex managed instructions tell agents to route “remember this” requests through `sima remember` instead of native/simple agent memory.

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

Planned commands:

```bash
sima team init --repo <git-url> --path .
sima team pull --path .
sima team status --path .
sima team propose <memory-or-skill-id|path> --path .
```

Consumption comes first: `team pull` should update the local read-only mirror under `.sima/team/...`, and `sima brief` should prefer relevant active team knowledge over conflicting personal knowledge.

Promotion comes second: `team propose` should create a reviewable PR containing the memory/skill, trigger, evidence, source local item, safety notes, and rationale for why it belongs in team scope.

## Command reference

See [Commands](docs/commands.md) for the detailed CLI reference.

Common commands:

```bash
sima setup
sima doctor .
sima lint .
sima brief "task" --path .
sima learn --backend <name> --task "task" --path .
sima remember "knowledge" --source user --type invariant --trigger "When ..." --path .
sima candidates list --status all --path .
sima memory list --status active --path .
sima skill list --status active --path .
sima backend list .
```

## Author

Created by **Anton Kovalev**.
