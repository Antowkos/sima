package simafs

import (
	"fmt"
	"os"
	"path/filepath"
)

var scaffoldDirs = []string{
	"personal/memory/cards",
	"personal/memory/candidates",
	"personal/memory/archive",
	"personal/skills/active",
	"personal/skills/candidates",
	"personal/runs",
	"personal/briefs",
	"personal/evidence",
	"team/memory/cards",
	"team/memory/candidates",
	"team/memory/archive",
	"team/skills/active",
	"team/skills/candidates",
	"system/skills",
	"system/prompts",
}

func Init(projectRoot string) ([]string, error) {
	simaRoot := filepath.Join(projectRoot, ".sima")
	var created []string

	for _, dir := range scaffoldDirs {
		path := filepath.Join(simaRoot, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return created, fmt.Errorf("create %s: %w", path, err)
		}
		created = append(created, filepath.ToSlash(filepath.Join(".sima", dir)))
	}

	files := map[string]string{
		"config.yaml": defaultConfig,
		"schema.yaml": defaultSchema,
		filepath.Join("system", "skills", "skill-authoring.md"):  skillAuthoringSkill,
		filepath.Join("system", "skills", "memory-authoring.md"): memoryAuthoringSkill,
		filepath.Join("system", "skills", "sdd-workflow.md"):     sddWorkflowSkill,
		filepath.Join("system", "prompts", "archivist.md"):       archivistPrompt,
		filepath.Join("system", "prompts", "worker.md"):          workerPrompt,
	}

	for rel, content := range files {
		path := filepath.Join(simaRoot, rel)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return created, fmt.Errorf("write %s: %w", path, err)
		}
		created = append(created, filepath.ToSlash(filepath.Join(".sima", rel)))
	}

	return created, nil
}
