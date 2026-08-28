package simafs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func installCodexSkills(projectRoot string) ([]string, error) {
	skills := map[string]string{
		codexSkillTargets["sima"]:          codexSIMASkill(),
		codexSkillTargets["sima-brief"]:    codexSIMABriefSkill(),
		codexSkillTargets["sima-remember"]: codexSIMARememberSkill(),
	}
	var written []string
	for rel, content := range skills {
		path := filepath.Join(projectRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return written, fmt.Errorf("create %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
			return written, fmt.Errorf("write %s: %w", path, err)
		}
		written = append(written, rel)
	}
	sort.Strings(written)
	return written, nil
}

func codexSIMASkill() string {
	return `---
name: sima
description: Use when the user invokes /sima, mentions SIMA flow, or wants a task run with project memory. Runs sima brief, normal repo work, verification, then sima learn for durable lessons.
---

# SIMA task flow

Use this skill when the user invokes /sima <task> or asks to run work through SIMA.

1. Treat the user's message after /sima as the task. If no task is provided, ask for it and stop.
2. Run: sima brief "<task>" --path .
3. Read the brief and use active SIMA memory/skills plus the current repository as context.
4. Do the normal repository workflow: inspect files, use git/gh when relevant, edit files, and run real verification.
5. Preserve evidence: changed files, test/build output, important decisions, and blockers.
6. After successful bounded work, run: sima learn --backend <backend-name> --task "<task>" --path .

Use the backend configured for this project. If no backend is obvious, run sima backend list . and pick the configured Claude/Codex profile. If no backend exists, tell the user exactly what is missing and do not fake learning.

Only let SIMA learn durable, reusable project knowledge. Do not store transient progress, secrets, credentials, raw logs, or PR/issue numbers as durable facts.`
}

func codexSIMABriefSkill() string {
	return `---
name: sima-brief
description: Use when the user invokes /sima-brief or asks for a SIMA project-memory briefing for a task.
---

# SIMA brief

Use this skill when the user invokes /sima-brief <task> or asks for a SIMA briefing.

1. Treat the user's message after /sima-brief as the task. If no task is provided, ask for it and stop.
2. Run: sima brief "<task>" --path .
3. Summarize only the briefing sections that matter for the task.
4. Do not paste unrelated raw history into the conversation.`
}

func codexSIMARememberSkill() string {
	return `---
name: sima-remember
description: Use when the user invokes /sima-remember or asks to save durable project knowledge through SIMA.
---

# SIMA remember

Use this skill when the user invokes /sima-remember <knowledge> or asks to remember/save/record project knowledge.

1. Treat the user's message after /sima-remember as the durable project knowledge. If no knowledge is provided, ask what should be remembered and stop.
2. Classify the knowledge as one of: decision, invariant, gotcha, workflow, guardrail, anti_pattern, or open_question.
3. Create a clear trigger beginning with "When ...".
4. Run: sima remember "<knowledge>" --source user --type <type> --trigger "When ..." --path .
5. Report the candidate or applied result path shown by SIMA.

Do not store secrets, credentials, tokens, transient task progress, raw logs, or soon-stale issue/PR numbers.`
}
