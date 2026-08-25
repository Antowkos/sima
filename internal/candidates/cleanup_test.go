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

func TestListAndShowCandidates(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatal(err)
	}
	candidateDir := filepath.Join(root, ".sima", "personal", "memory", "candidates")
	candidate := `version: 1
id: inspect-me
kind: run_reflection
scope: personal
operation: create
status: candidate
archivist_decision: apply
safety:
  decision: safe
learning:
  destination: memory
  operation: create
candidate_memories:
  - type: invariant
    title: Inspect candidates
    trigger: When candidate queues need review.
    summary: Candidate list and show expose proposal metadata before mutation.
run:
  id: run
  path: .sima/personal/runs/run
`
	if err := os.WriteFile(filepath.Join(candidateDir, "inspect-me.yaml"), []byte(candidate), 0o644); err != nil {
		t.Fatal(err)
	}
	deferred := strings.Replace(candidate, "id: inspect-me", "id: deferred", 1)
	deferred = strings.Replace(deferred, "status: candidate", "status: deferred", 1)
	if err := os.WriteFile(filepath.Join(candidateDir, "deferred.yaml"), []byte(deferred), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := List(root, ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "inspect-me" || items[0].Destination != "memory" || items[0].Candidates != 1 {
		t.Fatalf("unexpected candidate list: %+v", items)
	}
	all, err := List(root, ListOptions{Status: "all"})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected two all-status candidates, got %+v", all)
	}
	shown, err := Show(root, "inspect-me")
	if err != nil {
		t.Fatal(err)
	}
	if shown.Proposal.ID != "inspect-me" || !strings.Contains(shown.Content, "title: Inspect candidates") || shown.Path != ".sima/personal/memory/candidates/inspect-me.yaml" {
		t.Fatalf("unexpected shown candidate: %+v", shown)
	}
	outsidePath := filepath.Join(root, "outside.yaml")
	if err := os.WriteFile(outsidePath, []byte(candidate), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Show(root, outsidePath); err == nil {
		t.Fatal("Show should reject paths outside the candidate directory")
	}
}
