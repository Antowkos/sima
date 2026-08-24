package simafs

const defaultConfig = `version: 1
project:
  name: ""
  mode: personal-first
policy:
  personal_auto_apply: true
  team_auto_apply: false
  require_clean_archivist_session: true
  reject_reward_hacking: true
backends: {}
`

const defaultSchema = `version: 1
memory_types:
  - decision
  - invariant
  - gotcha
  - workflow
  - rejected_approach
  - open_question
  - anti_pattern
  - guardrail
proposal_operations:
  - create
  - update
  - deprecate
  - supersede
proposal_scopes:
  - personal
  - team
safety_decisions:
  - safe
  - suspicious
  - unsafe
archivist_decisions:
  - apply
  - reject
  - defer
`

const skillAuthoringSkill = `---
name: skill-authoring
description: "Use when creating or updating SIMA skills. Writes trigger-first reusable procedures with pitfalls and verification."
scope: system
managed_by: sima
---

# Skill Authoring

## Trigger

Use this when drafting, updating, deprecating, or superseding a skill.

## Rules

1. A skill is procedural memory: how to act repeatably, not just what happened.
2. Start with a clear trigger: "Use when ...".
3. Include steps, pitfalls, verification, and when not to use.
4. Avoid transient task status, PR numbers, raw logs, secrets, and one-off details.
5. Prefer updating an existing skill over creating a near-duplicate.
6. If a workflow is destructive, test-weakening, or reward-hacking, do not promote it as a skill; propose an anti-pattern or guardrail instead.
`

const memoryAuthoringSkill = `---
name: memory-authoring
description: "Use when creating or updating SIMA memory cards. Produces triggerable, evidence-backed knowledge."
scope: system
managed_by: sima
---

# Memory Authoring

## Trigger

Use this when drafting, updating, deprecating, or superseding memory cards.

## Rules

1. A memory card stores what is true, decided, risky, rejected, or important to recall.
2. Every useful memory needs a trigger: when should a future agent recall this?
3. Attach evidence pointers whenever possible.
4. Do not store transient progress, raw summaries, PR/issue status, secrets, or stale TODOs.
5. Choose the right operation: create, update, deprecate, or supersede.
6. Use anti_pattern or guardrail for lessons about what not to do.
`

const sddWorkflowSkill = `---
name: sdd-workflow
description: "Use when running SIMA with spec-driven development. Preserve specs as source artifacts and derive compact briefings."
scope: system
managed_by: sima
---

# SDD Workflow

## Trigger

Use this when a task has product specs, technical specs, OpenSpec changes, or implementation plans.

## Rules

1. Treat specs/plans as source artifacts, not active memory by default.
2. Brief from specs by extracting constraints, acceptance criteria, open questions, and verification gates.
3. Do not save the full spec as a memory card.
4. After execution, propose only durable lessons: decisions, invariants, gotchas, guardrails, anti-patterns, or reusable skills.
5. Preserve evidence pointers back to the exact spec/plan paths.
6. If implementation diverges from spec, record the divergence and require archivist scrutiny before learning from it.
`

const archivistPrompt = `# SIMA Archivist

You are the clean-session archivist/checker for SIMA. You did not perform the task. Judge only bounded evidence: task, brief, diff, logs, verification, worker report, proposals, and relevant existing memory/skills.

Responsibilities:
- detect conflicts, stale knowledge, bad evidence, duplicates, and unsafe changes;
- reject reward hacking: deleted inconvenient code, weakened tests, changed requirements, bypassed validation, hardcoded outputs, ignored checks, hid errors, or learned destructive lessons;
- decide create/update/deprecate/supersede for memory and skills;
- emit structured decisions only: apply, reject, or defer with reasons and evidence.

Never promote "green tests" as a lesson unless the task was solved without destroying requirements or verification quality.
`

const workerPrompt = `# SIMA Worker

Perform the bounded task using the provided brief. Preserve evidence. Do not weaken tests, bypass validation, change requirements, hardcode outputs, hide errors, or remove obstacles instead of fixing root causes.

Return YAML only. The first non-empty line must be ` + "`" + `proposed_memory:` + "`" + `, ` + "`" + `proposed_skills:` + "`" + `, or ` + "`" + `status:` + "`" + `. Proposed items must follow this contract:

` + "```yaml" + `
proposed_memory:
  - type: gotcha|workflow|decision|guardrail|invariant|anti_pattern
    title: Short durable lesson title
    trigger: When a future agent should recall this memory.
    summary: Evidence-backed lesson; no transient task progress.
proposed_skills:
  - name: lowercase-skill-name
    trigger: When a future agent should use this reusable workflow.
    summary: What the skill should teach or do.
` + "```" + `

Omit both lists if there is no durable lesson. Do not wrap the YAML in prose or Markdown fences.
`
