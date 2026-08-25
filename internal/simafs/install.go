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
`, label)
}
