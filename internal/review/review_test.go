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
	if item.ID == "" || item.RunID != runResult.RunID || item.Safety != "safe" || item.Destination != "session_only" || item.Operation != "create" || len(item.Problems) != 0 {
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

func TestReviewReportsLowQualityMemoryCandidate(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	path := filepath.Join(root, ".sima", "personal", "memory", "candidates", "low-quality.yaml")
	data := `version: 1
id: low-quality
kind: run_reflection
scope: personal
operation: create
status: candidate
archivist_decision: defer
safety:
  decision: safe
run:
  id: run-1
  path: .sima/personal/runs/run-1
  status: success
evidence:
  - kind: stdout
    path: .sima/personal/runs/run-1/stdout.log
candidate_source: structured
candidate_memories:
  - type: note
    title: Pushed today's fix
    trigger: Remember this
    summary: PR #123 was pushed today.
    evidence:
      - kind: stdout
        path: .sima/personal/runs/run-1/stdout.log
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Review(root, Options{})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	joined := strings.Join(result.Items[0].Problems, "\n")
	for _, want := range []string{"type must be", "trigger must describe when", "summary must be", "transient task progress"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("problems missing %q: %s", want, joined)
		}
	}
}

func TestReviewAcceptsPullRequestFormattingPolicy(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	result, err := proposal.Remember(root, proposal.RememberOptions{
		Text:    "User's explicit standing house style for every PR opened in this repo. PR title is TICKETKEY colon short description matching the commit summary. PR body opens with a level-2 markdown heading linking the Jira ticket and repeating the title, then a section titled Что сделано listing what was done. Never include Summary or Test plan template sections, checklists, emojis, or a bot or AI attribution footer.",
		Source:  "user",
		Type:    "invariant",
		Title:   "PR title and body format basics",
		Trigger: "When opening a pull request in this repo",
		Now:     time.Date(2026, 9, 1, 13, 5, 4, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Remember() error = %v", err)
	}
	reviewResult, err := Review(root, Options{})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if len(reviewResult.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(reviewResult.Items))
	}
	if len(reviewResult.Items[0].Problems) != 0 {
		t.Fatalf("PR-format policy should pass review validation; proposal=%s problems=%v", result.Path, reviewResult.Items[0].Problems)
	}
}
