package apply

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

func TestApplyPromotesSafeApprovedProposal(t *testing.T) {
	root, proposalPath := createProposal(t)
	approveProposal(t, proposalPath)

	result, err := Apply(root, Options{Target: proposalPath, Now: time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("applied = %d, want 1", len(result.Applied))
	}
	if !strings.HasPrefix(result.Applied[0], ".sima/personal/memory/cards/") {
		t.Fatalf("unexpected applied path: %s", result.Applied[0])
	}
	cardData, err := os.ReadFile(filepath.Join(root, result.Applied[0]))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cardData), "status: active") || !strings.Contains(string(cardData), "source:") {
		t.Fatalf("unexpected card:\n%s", cardData)
	}
	proposalData, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(proposalData), "status: applied") || !strings.Contains(string(proposalData), "applied_at:") {
		t.Fatalf("proposal not marked applied:\n%s", proposalData)
	}
}

func TestApplyRequiresArchivistApply(t *testing.T) {
	root, proposalPath := createProposal(t)
	_, err := Apply(root, Options{Target: proposalPath})
	if err == nil || !strings.Contains(err.Error(), "archivist_decision must be apply") {
		t.Fatalf("expected archivist gate error, got %v", err)
	}
}

func createProposal(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := config.AddBackend(root, "echo", config.BackendProfile{Kind: "codex", Executable: "/bin/echo"}, false); err != nil {
		t.Fatalf("AddBackend() error = %v", err)
	}
	runResult, err := runner.Run(root, runner.Options{BackendName: "echo", Task: "apply proposal", Now: time.Date(2026, 8, 23, 11, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	reportPath := filepath.Join(root, ".sima", "personal", "runs", runResult.RunID, "worker-report.yaml")
	report := "run_id: " + runResult.RunID + "\nstatus: success\nexit_code: 0\ntask: apply proposal\nproposed_memory:\n  - type: workflow\n    title: Apply approved structured proposal\n    trigger: When a safe structured SIMA proposal has archivist approval.\n    summary: SIMA apply promotes approved structured personal proposals into active memory cards with evidence.\n"
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	proposalResult, err := proposal.Generate(root, proposal.Options{FromRun: runResult.RunID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	return root, proposalResult.Path
}

func approveProposal(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "archivist_decision: defer", "archivist_decision: apply", 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
}
