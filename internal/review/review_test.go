package review

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antowkos/sima/internal/config"
	"github.com/antowkos/sima/internal/proposal"
	"github.com/antowkos/sima/internal/runner"
	"github.com/antowkos/sima/internal/simafs"
)

func TestReviewListsValidCandidate(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := config.AddBackend(root, "echo", config.BackendProfile{Kind: "codex", Executable: "/bin/echo"}, false); err != nil {
		t.Fatalf("AddBackend() error = %v", err)
	}
	runResult, err := runner.Run(root, runner.Options{BackendName: "echo", Task: "review candidate", Now: time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := proposal.Generate(root, proposal.Options{FromRun: runResult.RunID}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	result, err := Review(root, Options{})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.ID == "" || item.RunID != runResult.RunID || item.Safety != "safe" || len(item.Problems) != 0 {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestReviewReportsInvalidProposal(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	path := filepath.Join(root, ".sima", "personal", "memory", "candidates", "bad.yaml")
	if err := os.WriteFile(path, []byte("id: bad\nstatus: candidate\nsafety:\n  decision: unsafe\narchivist_decision: apply\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Review(root, Options{})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(result.Items))
	}
	joined := strings.Join(result.Items[0].Problems, "\n")
	for _, want := range []string{"missing version", "missing evidence", "suspicious/unsafe proposals cannot be apply"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("problems missing %q: %s", want, joined)
		}
	}
}
