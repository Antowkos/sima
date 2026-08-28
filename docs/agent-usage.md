# Using SIMA with Agents

SIMA is agent-first: humans install and configure it, but Claude Code, Codex, and similar coding agents are the primary users during day-to-day work.

The mental model is:

```text
agent task → sima brief → normal repo work → verification → sima learn/remember → better future agent context
```

SIMA does not replace an agent's normal tools. The agent should still inspect the repository, use `git`/`gh`, edit files, run tests, and report real evidence. SIMA adds durable project memory and reusable skills around that workflow.

## What gets installed where

Run from a project root:

```bash
sima setup
```

or separately:

```bash
sima init .
sima install --client all --path .
sima backend add <name> --kind <claude-code|codex> --executable <path> --path .
```

SIMA writes project-local state, managed agent instructions, and Claude Code slash commands:

```text
.sima/                         # project-local memory, skills, runs, evidence, config
CLAUDE.md                      # Claude Code project instructions
AGENTS.md                      # Codex/OpenAI agent project instructions
.claude/commands/sima.md       # Claude: /sima <task>
.claude/commands/sima-brief.md # Claude: /sima-brief <task>
.claude/commands/sima-remember.md # Claude: /sima-remember <knowledge>
```

`CLAUDE.md` and `AGENTS.md` are upserted with a managed block:

```html
<!-- BEGIN SIMA MANAGED INSTRUCTIONS -->
...
<!-- END SIMA MANAGED INSTRUCTIONS -->
```

Content outside that block is preserved. Re-running `sima install` or `sima setup` updates only the managed block.

## What the managed instructions tell agents

The installed Claude/Codex block tells the agent to:

1. run `sima brief "<task>" --path .` before starting meaningful work;
2. use active SIMA memory/skills plus the current repository as context;
3. keep raw logs, secrets, credentials, and unrelated history out of memory;
4. preserve evidence: tests, build output, changed files, decisions;
5. route explicit “remember/save/learn/record this” requests through `sima remember`;
6. use `sima learn` after successful bounded work so durable lessons can be proposed and reviewed;
7. avoid learning transient task progress, raw run summaries, PR numbers, tokens, credentials, or stale TODOs.

## How to use SIMA in Claude Code

After `sima setup`, Claude Code reads `CLAUDE.md` automatically for the project. SIMA also installs project slash commands under `.claude/commands/`.

You can type:

```text
/sima fix the failing auth tests
/sima-brief plan the database migration
/sima-remember API handlers must use generated request types
```

Command behavior:

- `/sima <task>`: runs `sima brief`, performs the normal repo workflow, verifies, then runs `sima learn` for durable lessons.
- `/sima-brief <task>`: only generates and summarizes a SIMA briefing.
- `/sima-remember <knowledge>`: classifies durable project knowledge and routes it through `sima remember`.

Suggested human prompt when not using slash commands:

```text
Use SIMA for this task. Start with `sima brief`, do the normal repo workflow, verify with tests, then run `sima learn` only for durable lessons.

Task: <your task>
```

Typical Claude-side commands:

```bash
sima brief "<task>" --path .
# inspect repo, edit files, run tests/builds normally
sima learn --backend claude-main --task "<task>" --path .
```

If Claude is using a separate config/account, configure it during setup:

```bash
sima setup --backend claude --claude-config-dir ~/.claude-work
```

or with an explicit wrapper:

```bash
sima setup --backend claude --executable /path/to/claude-wrapper
```

## How to use SIMA in Codex

After `sima setup`, Codex reads `AGENTS.md` automatically for the project.

Current Codex support is instruction-based rather than confirmed project slash-command based: SIMA tells Codex to route SIMA-like requests through `sima brief`, `sima learn`, and `sima remember`. Do not rely on unknown Codex slash commands being accepted by the TUI until SIMA adds a verified Codex plugin/command integration.

Suggested human prompt:

```text
Use SIMA for this task. Start with `sima brief`, do normal repo/tool work, run verification, then let SIMA capture durable lessons.

Task: <your task>
```

Typical Codex-side commands:

```bash
sima brief "<task>" --path .
# inspect repo, edit files, run tests/builds normally
sima learn --backend codex-main --task "<task>" --path .
```

For editable Codex tasks, SIMA should store a workspace-write sandbox mode:

```bash
sima backend add codex-main \
  --kind codex \
  --executable "$(command -v codex)" \
  --permission-mode workspace-write \
  --path .
```

SIMA then invokes Codex as:

```bash
codex exec --sandbox workspace-write ...
```

Before first real use, verify Codex auth outside SIMA:

```bash
codex doctor
# if needed: codex login
```

## Two ways agents use SIMA

### 1. Current-session assist

Use this when the agent is already working in Claude/Codex and should keep control of the task:

```bash
sima brief "<task>" --path .
# agent does the work normally
sima remember "<durable knowledge>" --source user|review|agent --type <type> --trigger "When ..." --path .
```

This is best for investigation, reviews, answering questions, and explicit memory requests.

### 2. SIMA-managed worker run

Use this when you want SIMA to run a bounded task through a configured backend and capture evidence automatically:

```bash
sima learn --backend <backend-name> --task "<task>" --path .
```

This is best for implementation tasks, PR-fix tasks, dogfood runs, and repeatable automation. `sima learn --json` is preferred for wrappers.

## Review and PR workflows

For normal review requests such as:

```text
Look at PR comments and tell me what matters.
```

the agent should use its normal workflow first:

```bash
gh pr view <pr>
gh pr checks <pr>
gh pr diff <pr>
# inspect files, comments, CI, tests
```

Only after a durable lesson is clear should it call:

```bash
sima remember "<lesson>" --source review --type <type> --trigger "When ..." --path .
```

For implementation requests such as:

```text
Fix/address PR comments through SIMA.
```

the managed instructions route the implementation to:

```bash
sima learn --backend <backend-name> \
  --task "Address PR review comments using gh/repo inspection, implement fixes, run verification, and propose durable lessons only if found." \
  --path .
```

The worker still uses the normal GitHub/repo workflow inside the SIMA run.

## Team flow

Team knowledge is promoted, not born shared:

```text
local learn → personal active memory/skill → explicit team proposal → PR review → merge → team pull → future briefs
```

Planned commands:

```bash
sima team init --repo <git-url> --path .
sima team pull --path .
sima team status --path .
sima team propose <memory-or-skill-id|path> --path .
```

The consumption side comes first: `team pull` should update `.sima/team/...`, and `sima brief` should prefer relevant active team knowledge over conflicting personal knowledge. Promotion comes later through a reviewable PR into the shared team knowledge repository.

## Safety rules for agents

Agents should not use SIMA to store:

- API keys, tokens, credentials, or connection strings;
- raw chat logs or raw terminal output;
- transient task progress;
- PR/issue numbers as durable facts;
- unverifiable claims;
- one-off summaries pretending to be reusable skills.

Good SIMA memory is compact, triggerable, evidence-backed, and useful in a future task. Good SIMA skills are reusable workflows with steps, pitfalls, and verification.
