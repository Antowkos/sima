package proposal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antowkos/sima/internal/config"
	"github.com/antowkos/sima/internal/runner"
	"github.com/antowkos/sima/internal/simafs"
)

func TestGenerateCreatesRunProposal(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := config.AddBackend(root, "echo", config.BackendProfile{Kind: "codex", Executable: "/bin/echo"}, false); err != nil {
		t.Fatalf("AddBackend() error = %v", err)
	}
	runResult, err := runner.Run(root, runner.Options{
		BackendName: "echo",
		Task:        "capture proposal evidence",
		Now:         time.Date(2026, 8, 22, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result, err := Generate(root, Options{FromRun: runResult.RunID, Now: time.Date(2026, 8, 22, 13, 1, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.RunID != runResult.RunID {
		t.Fatalf("RunID = %q, want %q", result.RunID, runResult.RunID)
	}
	if result.Safety != "safe" || result.Decision != "defer" || result.Candidates != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"kind: run_reflection", "archivist_decision: defer", "decision: safe", "worker-report.yaml"} {
		if !strings.Contains(text, want) {
			t.Fatalf("proposal missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateLastRunAndSafetyFlags(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	runID := "20260822-130000-manual"
	runDir := filepath.Join(root, ".sima", "personal", "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "worker-report.yaml"), []byte("run_id: "+runID+"\nstatus: success\nexit_code: 0\ntask: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "stdout.log"), []byte("hardcoded output and skipped validation\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{FromRun: "last"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Safety != "suspicious" || result.Candidates != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "decision: suspicious") {
		t.Fatalf("expected suspicious proposal:\n%s", data)
	}
}
