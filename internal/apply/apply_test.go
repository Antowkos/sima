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
	"gopkg.in/yaml.v3"
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

func TestApplyUpdatesTargetMemory(t *testing.T) {
	root, proposalPath := createProposal(t)
	target := writeExistingMemory(t, root, "existing", "Old title", "Old summary")
	mutateProposal(t, proposalPath, func(p *proposal.Proposal) {
		p.ArchivistDecision = "apply"
		p.Learning.Operation = "update"
		p.Learning.Target = proposal.LearningTarget{Kind: "memory", Path: target, ID: "existing"}
	})

	result, err := Apply(root, Options{Target: proposalPath, Now: time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(result.Applied) != 1 || result.Applied[0] != target {
		t.Fatalf("applied = %#v, want target %s", result.Applied, target)
	}
	data, err := os.ReadFile(filepath.Join(root, target))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "title: Apply approved structured proposal") || !strings.Contains(text, "status: active") || !strings.Contains(text, "proposal_id:") {
		t.Fatalf("target memory not updated:\n%s", text)
	}
}

func TestApplySupersedesTargetMemory(t *testing.T) {
	root, proposalPath := createProposal(t)
	target := writeExistingMemory(t, root, "existing", "Old title", "Old summary")
	mutateProposal(t, proposalPath, func(p *proposal.Proposal) {
		p.ArchivistDecision = "apply"
		p.Learning.Operation = "supersede"
		p.Learning.Target = proposal.LearningTarget{Kind: "memory", Path: target, ID: "existing"}
	})

	result, err := Apply(root, Options{Target: proposalPath, Now: time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(result.Applied) != 2 {
		t.Fatalf("applied = %#v, want old target plus new card", result.Applied)
	}
	oldData, err := os.ReadFile(filepath.Join(root, target))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(oldData), "status: superseded") {
		t.Fatalf("target not superseded:\n%s", oldData)
	}
	newData, err := os.ReadFile(filepath.Join(root, result.Applied[1]))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(newData), "status: active") || !strings.Contains(string(newData), "Apply approved structured proposal") {
		t.Fatalf("new card not created:\n%s", newData)
	}
}

func TestApplyDeprecatesTargetMemory(t *testing.T) {
	root, proposalPath := createProposal(t)
	target := writeExistingMemory(t, root, "existing", "Old title", "Old summary")
	mutateProposal(t, proposalPath, func(p *proposal.Proposal) {
		p.ArchivistDecision = "apply"
		p.Learning.Operation = "deprecate"
		p.Learning.Target = proposal.LearningTarget{Kind: "memory", Path: target, ID: "existing"}
	})

	result, err := Apply(root, Options{Target: proposalPath, Now: time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(result.Applied) != 1 || result.Applied[0] != target {
		t.Fatalf("applied = %#v, want target %s", result.Applied, target)
	}
	data, err := os.ReadFile(filepath.Join(root, target))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "status: deprecated") {
		t.Fatalf("target not deprecated:\n%s", data)
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

func mutateProposal(t *testing.T, path string, mutate func(*proposal.Proposal)) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var p proposal.Proposal
	if err := yaml.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	mutate(&p)
	updated, err := yaml.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeExistingMemory(t *testing.T, root, id, title, summary string) string {
	t.Helper()
	relPath := filepath.Join(".sima", "personal", "memory", "cards", id+".yaml")
	absPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	card := ActiveMemoryCard{
		ID:        id,
		Type:      "workflow",
		Title:     title,
		Trigger:   "When testing operation-aware apply.",
		Summary:   summary,
		Status:    "active",
		Scope:     "personal",
		CreatedAt: "2026-08-22T12:00:00Z",
		UpdatedAt: "2026-08-22T12:00:00Z",
	}
	data, err := yaml.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(relPath)
}
