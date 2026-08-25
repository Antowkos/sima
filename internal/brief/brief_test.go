package brief

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antowkos/sima/internal/simafs"
)

func TestGenerateWritesBriefWithSddArtifacts(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	planPath := filepath.Join(root, "docs", "plans", "test-plan.md")
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, []byte("# Test Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	memoryPath := filepath.Join(root, ".sima", "personal", "memory", "cards", "gotcha.yaml")
	if err := os.WriteFile(memoryPath, []byte("id: gotcha\nstatus: active\ntype: gotcha\ntitle: Remember active cards\ntrigger: When building a SIMA brief\nsummary: Active memory content should appear in the brief.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deprecatedMemoryPath := filepath.Join(root, ".sima", "personal", "memory", "cards", "deprecated.yaml")
	if err := os.WriteFile(deprecatedMemoryPath, []byte("id: deprecated\nstatus: deprecated\ntitle: Deprecated card\nsummary: Deprecated memory content must not appear in the brief.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(root, ".sima", "personal", "skills", "active", "brief-skill.md")
	if err := os.WriteFile(skillPath, []byte("---\nstatus: active\n---\n# Brief Skill\n\nUse when testing active skill snippets.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	supersededSkillPath := filepath.Join(root, ".sima", "personal", "skills", "active", "old-skill.md")
	if err := os.WriteFile(supersededSkillPath, []byte("---\nstatus: superseded\n---\n# Old Skill\n\nSuperseded skill content must not appear in the brief.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{Task: "build brief", Now: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("brief file missing: %v", err)
	}
	for _, want := range []string{"# SIMA Brief", "build brief", ".sima/system/skills/sdd-workflow.md", ".sima/personal/memory/cards/gotcha.yaml", "Remember active cards", "Active memory content should appear in the brief", ".sima/personal/skills/active/brief-skill.md", "Use when testing active skill snippets", "docs/plans/test-plan.md"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("brief missing %q:\n%s", want, result.Content)
		}
	}
	for _, unwanted := range []string{"Deprecated card", "Deprecated memory content must not appear", ".sima/personal/memory/cards/deprecated.yaml", "Old Skill", "Superseded skill content must not appear", ".sima/personal/skills/active/old-skill.md"} {
		if strings.Contains(result.Content, unwanted) {
			t.Fatalf("brief included inactive item %q:\n%s", unwanted, result.Content)
		}
	}
}

func TestGenerateRequiresTask(t *testing.T) {
	_, err := Generate(t.TempDir(), Options{})
	if err == nil {
		t.Fatal("expected missing task error")
	}
}
