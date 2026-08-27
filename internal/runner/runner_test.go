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

func TestRunAllocatesUniqueRunDir(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := config.AddBackend(root, "echo", config.BackendProfile{Kind: "codex", Executable: "/bin/echo"}, false); err != nil {
		t.Fatalf("AddBackend() error = %v", err)
	}
	now := time.Date(2026, 8, 22, 12, 31, 0, 0, time.UTC)
	first, err := Run(root, Options{BackendName: "echo", Task: "first", Now: now})
	if err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	second, err := Run(root, Options{BackendName: "echo", Task: "second", Now: now})
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if first.RunID == second.RunID || first.RunDir == second.RunDir {
		t.Fatalf("run ids should be unique: first=%+v second=%+v", first, second)
	}
	if !strings.HasSuffix(second.RunID, "-02") {
		t.Fatalf("second RunID = %q, want collision suffix", second.RunID)
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

func TestBuildArgsAddsClaudeJSONSchemaMode(t *testing.T) {
	profile := config.BackendProfile{Kind: "claude-code", Metadata: map[string]string{"output_format": "json_schema"}}
	args := buildArgs(profile, "do task")
	want := []string{"-p", "--output-format", "json", "--json-schema"}
	for i, value := range want {
		if args[i] != value {
			t.Fatalf("args[%d] = %q, want %q; args=%v", i, args[i], value, args)
		}
	}
	if !strings.Contains(args[4], "proposed_memory") || !strings.Contains(args[4], "enum") || !strings.Contains(args[4], "open_question") {
		t.Fatalf("json schema missing strict proposal fields: %s", args[4])
	}
	if args[len(args)-1] != "do task" {
		t.Fatalf("prompt arg = %q, want do task; args=%v", args[len(args)-1], args)
	}
}

func TestBuildArgsAddsCodexSandboxPermissionMode(t *testing.T) {
	profile := config.BackendProfile{Kind: "codex", PermissionMode: "workspace-write"}
	args := buildArgs(profile, "do task")
	want := []string{"exec", "--sandbox", "workspace-write", "do task"}
	if len(args) != len(want) {
		t.Fatalf("args len = %d, want %d; args=%v", len(args), len(want), args)
	}
	for i, value := range want {
		if args[i] != value {
			t.Fatalf("args[%d] = %q, want %q; args=%v", i, args[i], value, args)
		}
	}
}
