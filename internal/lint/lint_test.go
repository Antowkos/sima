package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antowkos/sima/internal/simafs"
)

func TestCheckReportsInvalidStatusAndUnsafeTarget(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatal(err)
	}
	memoryDir := filepath.Join(root, ".sima", "personal", "memory", "cards")
	if err := os.WriteFile(filepath.Join(memoryDir, "bad.yaml"), []byte("id: bad\nstatus: stale\ntitle: Bad status\nsummary: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	candidateDir := filepath.Join(root, ".sima", "personal", "memory", "candidates")
	candidate := `version: 1
id: unsafe
kind: memory
scope: personal
operation: deprecate
status: candidate
archivist_decision: defer
safety:
  decision: safe
run:
  id: run
  path: .sima/personal/runs/run
learning:
  destination: memory
  operation: deprecate
  target:
    kind: memory
    path: ../outside.yaml
evidence:
  - kind: stdout
    path: .sima/personal/runs/run/stdout.log
`
	if err := os.WriteFile(filepath.Join(candidateDir, "unsafe.yaml"), []byte(candidate), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorCount() != 2 || result.WarningCount() != 1 {
		t.Fatalf("unexpected counts: errors=%d warnings=%d issues=%v", result.ErrorCount(), result.WarningCount(), result.Issues)
	}
	joined := joinIssues(result.Issues)
	for _, want := range []string{"status must be active", "learning.target.path resolves outside project root", "pending candidate"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("lint output missing %q:\n%s", want, joined)
		}
	}
}

func TestCheckRequiresExplicitStatus(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatal(err)
	}
	memoryDir := filepath.Join(root, ".sima", "personal", "memory", "cards")
	if err := os.WriteFile(filepath.Join(memoryDir, "missing-status.yaml"), []byte("id: missing\ntype: workflow\ntitle: Missing status\ntrigger: When checking cards before release.\nsummary: Cards must carry explicit lifecycle status during active product development.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorCount() != 1 || result.WarningCount() != 0 {
		t.Fatalf("expected one missing-status error, got %+v", result.Issues)
	}
	if !strings.Contains(joinIssues(result.Issues), "status is required") {
		t.Fatalf("expected missing status issue, got %+v", result.Issues)
	}
}

func joinIssues(issues []Issue) string {
	var b strings.Builder
	for _, issue := range issues {
		b.WriteString(issue.Severity)
		b.WriteString(" ")
		b.WriteString(issue.Path)
		b.WriteString(" ")
		b.WriteString(issue.Message)
		b.WriteString("\n")
	}
	return b.String()
}
