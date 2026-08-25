package candidates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antowkos/sima/internal/simafs"
)

func TestCleanupDeferredMarksPendingDeferredCandidates(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatal(err)
	}
	candidateDir := filepath.Join(root, ".sima", "personal", "memory", "candidates")
	candidatePath := filepath.Join(candidateDir, "deferred.yaml")
	candidate := `version: 1
id: deferred
kind: run_reflection
scope: personal
operation: create
status: candidate
archivist_decision: defer
safety:
  decision: safe
run:
  id: run
  path: .sima/personal/runs/run
`
	if err := os.WriteFile(candidatePath, []byte(candidate), 0o644); err != nil {
		t.Fatal(err)
	}
	applyPath := filepath.Join(candidateDir, "apply.yaml")
	applyCandidate := strings.Replace(candidate, "id: deferred", "id: apply", 1)
	applyCandidate = strings.Replace(applyCandidate, "archivist_decision: defer", "archivist_decision: apply", 1)
	if err := os.WriteFile(applyPath, []byte(applyCandidate), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := CleanupDeferred(root, CleanupOptions{Now: time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Updated) != 1 || result.Updated[0] != ".sima/personal/memory/candidates/deferred.yaml" {
		t.Fatalf("unexpected cleanup result: %+v", result)
	}
	data, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"status: deferred", "cleanup_at: \"2026-08-25T10:00:00Z\"", "cleanup_note: deferred pending candidate cleaned from active review queue"} {
		if !strings.Contains(text, want) {
			t.Fatalf("cleaned candidate missing %q:\n%s", want, text)
		}
	}
	applyData, err := os.ReadFile(applyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(applyData), "status: candidate") {
		t.Fatalf("apply candidate should remain pending:\n%s", applyData)
	}
}
