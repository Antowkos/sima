package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antowkos/sima/internal/config"
	"github.com/antowkos/sima/internal/simafs"
)

func TestRunCreatesArtifacts(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := config.AddBackend(root, "echo", config.BackendProfile{Kind: "codex", Executable: "/bin/echo"}, false); err != nil {
		t.Fatalf("AddBackend() error = %v", err)
	}

	result, err := Run(root, Options{
		BackendName: "echo",
		Task:        "test artifact capture",
		Now:         time.Date(2026, 8, 22, 12, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d", result.ExitCode)
	}
	for _, rel := range []string{"task.md", "brief.md", "command.txt", "stdout.log", "stderr.log", "worker-report.yaml"} {
		path := filepath.Join(result.RunDir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	stdoutBytes, err := os.ReadFile(filepath.Join(result.RunDir, "stdout.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stdoutBytes), "test artifact capture") {
		t.Fatalf("stdout missing task: %s", stdoutBytes)
	}
	reportBytes, err := os.ReadFile(filepath.Join(result.RunDir, "worker-report.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(reportBytes), "status: success") {
		t.Fatalf("unexpected report: %s", reportBytes)
	}
}

func TestRunRequiresConfiguredBackend(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatal(err)
	}
	_, err := Run(root, Options{BackendName: "missing", Task: "x"})
	if err == nil {
		t.Fatal("expected missing backend error")
	}
}
