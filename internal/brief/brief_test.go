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
	if err := os.WriteFile(memoryPath, []byte("id: gotcha\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{Task: "build brief", Now: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("brief file missing: %v", err)
	}
	for _, want := range []string{"# SIMA Brief", "build brief", ".sima/system/skills/sdd-workflow.md", ".sima/personal/memory/cards/gotcha.yaml", "docs/plans/test-plan.md"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("brief missing %q:\n%s", want, result.Content)
		}
	}
}

func TestGenerateRequiresTask(t *testing.T) {
	_, err := Generate(t.TempDir(), Options{})
	if err == nil {
		t.Fatal("expected missing task error")
	}
}
