package simafs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const managedBlockStart = "<!-- BEGIN SIMA MANAGED INSTRUCTIONS -->"
const managedBlockEnd = "<!-- END SIMA MANAGED INSTRUCTIONS -->"

var instructionTargets = map[string]string{
	"claude": "CLAUDE.md",
	"codex":  "AGENTS.md",
}

type InstallOptions struct {
	Clients []string
}

type InstallResult struct {
	Written []string
}

func InstallInstructions(projectRoot string, opts InstallOptions) (InstallResult, error) {
	clients := opts.Clients
	if len(clients) == 0 {
		clients = []string{"claude", "codex"}
	}
	var result InstallResult
	for _, client := range clients {
		normalized := strings.ToLower(strings.TrimSpace(client))
		filename, ok := instructionTargets[normalized]
		if !ok {
			return result, fmt.Errorf("unknown client %q", client)
		}
		path := filepath.Join(projectRoot, filename)
		if err := upsertManagedBlock(path, managedInstructions(normalized)); err != nil {
			return result, err
		}
		result.Written = append(result.Written, filepath.ToSlash(filename))
	}
	return result, nil
}

func upsertManagedBlock(path string, block string) error {
	existingBytes, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	existing := string(existingBytes)
	managed := managedBlockStart + "\n" + strings.TrimSpace(block) + "\n" + managedBlockEnd
	var next string
	start := strings.Index(existing, managedBlockStart)
	end := strings.Index(existing, managedBlockEnd)
	if start >= 0 && end >= start {
		end += len(managedBlockEnd)
		next = strings.TrimRight(existing[:start], " \t\r\n") + "\n\n" + managed + strings.TrimLeft(existing[end:], " \t\r\n")
	} else if strings.TrimSpace(existing) == "" {
		next = managed + "\n"
	} else {
		next = strings.TrimRight(existing, " \t\r\n") + "\n\n" + managed + "\n"
	}
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func managedInstructions(client string) string {
	label := "agent"
	if client == "claude" {
		label = "Claude Code"
	}
	if client == "codex" {
		label = "Codex"
	}
	return fmt.Sprintf(`# SIMA Project Memory Instructions for %s

SIMA is the project-local self-improvement memory harness for this repository.

## Before starting a task

1. Run `+"`"+`sima brief "<task>" --path .`+"`"+`.
2. Read the generated brief and use only active SIMA memory/skills plus the current repository as context.
3. Do not paste raw logs, secrets, credentials, or unrelated history into memory.

## During the task

- Preserve evidence: tests, build output, changed files, and important decisions.
- Do not weaken tests, bypass validation, hardcode outputs, hide errors, or change requirements to make the task look successful.
- Keep raw artifacts on disk; keep durable memory compact and triggerable.

## Explicit memory requests

If the user asks to "remember", "save", "learn", or "record" project knowledge, route it through the SIMA harness instead of %s native memory. Do not use the agent's built-in/simple memory for project knowledge covered by SIMA.

`+"```bash"+`
sima remember "<durable project knowledge>" --source user --type <decision|invariant|gotcha|workflow|guardrail|anti_pattern|open_question> --trigger "When ..." --path .
`+"```"+`

If an archivist backend is configured and the knowledge is safe to review immediately, include `+"`"+`--backend <backend-name>`+"`"+` so SIMA can run the clean archivist/apply flow. Otherwise leave the candidate pending and tell the user the proposal path.

Do not learn transient task progress, secrets, credentials, tokens, or raw chat history. If the user asks to remember a reusable procedure, prefer a SIMA skill candidate when that command exists; until then capture it as `+"`"+`--type workflow`+"`"+` or `+"`"+`--type guardrail`+"`"+` with a clear trigger.

## Review / investigation workflows

For normal review or investigation requests, such as "look at PR comments", first complete the normal tool workflow: use `+"`"+`gh`+"`"+`/repo inspection/checks/diff, answer or implement the review, and preserve evidence. Only after that, if a durable lesson was discovered, run `+"`"+`sima remember ... --source review --path .`+"`"+`. SIMA should learn from the completed evidence; it must not shortcut the familiar GitHub/repo workflow.

## SIMA-managed PR fixes

If the user asks to fix PR comments through SIMA, delegate the implementation to the harness instead of doing the edits directly in the current agent session:

`+"```bash"+`
sima learn --backend <backend-name> --task "Address PR review comments using gh/repo inspection, implement fixes, run verification, and propose durable lessons only if found." --path .
`+"```"+`

The SIMA worker should still use the normal GitHub/repo workflow inside the task: `+"`"+`gh pr view`+"`"+`, review/comment APIs, checks, diffs, file inspection, edits, and tests. Use this path when the request is to actually change code or "fix/address PR comments". For inspect-only requests like "look at PR comments" or "summarize review", do the normal investigation in the current session and use `+"`"+`sima remember ... --source review`+"`"+` only after a durable lesson is clear.

## After a successful task

Run SIMA learning for the completed task:

`+"```bash"+`
sima learn --backend <backend-name> --task "<task>" --path .
`+"```"+`

For wrappers or automation, prefer machine-readable output:

`+"```bash"+`
sima learn --backend <backend-name> --task "<task>" --json --path .
`+"```"+`

Use `+"`"+`--no-auto-apply`+"`"+` only for sensitive repos, demos, or debugging. Personal/local learning auto-applies by default only after archivist and apply-ready gates pass. Team/shared knowledge remains review-required.

## What SIMA should learn

Good memory: durable decisions, invariants, gotchas, guardrails, anti-patterns, open questions, and workflows with clear recall triggers and evidence.

Good skills: reusable procedures with trigger, steps, pitfalls, and verification.

Do not learn: transient task progress, raw run summaries, PR/issue numbers, secrets, credentials, tokens, or stale TODOs.
`, label, label)
}
