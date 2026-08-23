package archivist

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/antowkos/sima/internal/config"
	"github.com/antowkos/sima/internal/proposal"
	"github.com/antowkos/sima/internal/runner"
	"github.com/antowkos/sima/internal/simafs"
)

func TestDecideApprovesSafeValidPersonalProposal(t *testing.T) {
	root, proposalPath := createProposal(t, "archive safe proposal", false)

	result, err := Decide(root, Options{Target: proposalPath, Now: time.Date(2026, 8, 23, 13, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if result.Decision != "apply" {
		t.Fatalf("Decision = %q, notes = %v", result.Decision, result.Notes)
	}
	data, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "archivist_decision: apply") || !strings.Contains(text, "archivist_at:") || !strings.Contains(text, "deterministic archivist approved") {
		t.Fatalf("proposal not updated as apply:\n%s", text)
	}
}

func TestDecideRejectsSuspiciousProposal(t *testing.T) {
	root, proposalPath := createProposal(t, "archive suspicious proposal", true)

	result, err := Decide(root, Options{Target: proposalPath})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if result.Decision != "reject" {
		t.Fatalf("Decision = %q, notes = %v", result.Decision, result.Notes)
	}
	data, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "archivist_decision: reject") {
		t.Fatalf("proposal not rejected:\n%s", data)
	}
}

func createProposal(t *testing.T, task string, suspicious bool) (string, string) {
	t.Helper()
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	executable := "/bin/echo"
	if suspicious {
		executable = "/bin/sh"
	}
	kind := "codex"
	if suspicious {
		kind = "unknown"
	}
	_ = kind
	if suspicious {
		if err := config.AddBackend(root, "worker", config.BackendProfile{Kind: "claude-code", Executable: executable}, false); err != nil {
			t.Fatalf("AddBackend() error = %v", err)
		}
	} else {
		if err := config.AddBackend(root, "worker", config.BackendProfile{Kind: "codex", Executable: executable}, false); err != nil {
			t.Fatalf("AddBackend() error = %v", err)
		}
	}
	runResult, err := runner.Run(root, runner.Options{BackendName: "worker", Task: task, Now: time.Date(2026, 8, 23, 12, 30, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if suspicious {
		stdoutPath := root + "/.sima/personal/runs/" + runResult.RunID + "/stdout.log"
		if err := os.WriteFile(stdoutPath, []byte("hardcoded output and skipped validation\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	proposalResult, err := proposal.Generate(root, proposal.Options{FromRun: runResult.RunID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	return root, proposalResult.Path
}
