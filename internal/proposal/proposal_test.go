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
	for _, want := range []string{"kind: run_reflection", "archivist_decision: defer", "decision: safe", "destination: session_only", "fallback candidates require human/librarian review", "worker-report.yaml"} {
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

func TestGenerateUsesStructuredProposalsFromWorkerReport(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	runID := "20260822-140000-structured"
	runDir := filepath.Join(root, ".sima", "personal", "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	report := `run_id: 20260822-140000-structured
status: success
exit_code: 0
task: structured proposals
proposed_memory:
  - type: gotcha
    title: Structured proposals are parsed
    trigger: When a worker report contains proposed_memory.
    summary: SIMA should preserve structured worker-proposed memory instead of replacing it with fallback text.
proposed_skills:
  - name: structured-proposal-skill
    trigger: When a worker report contains proposed_skills.
    summary: Convert worker-proposed skills into candidate skills with evidence.
`
	if err := os.WriteFile(filepath.Join(runDir, "worker-report.yaml"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "stdout.log"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{FromRun: runID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Candidates != 2 {
		t.Fatalf("Candidates = %d, want 2", result.Candidates)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"Structured proposals are parsed", "structured-proposal-skill", "destination: mixed", "worker-report.yaml"} {
		if !strings.Contains(text, want) {
			t.Fatalf("proposal missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateRejectsYAMLStructuredStdout(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	runID := "20260822-150000-stdout"
	runDir := filepath.Join(root, ".sima", "personal", "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "worker-report.yaml"), []byte("run_id: "+runID+"\nstatus: success\nexit_code: 0\ntask: stdout proposals\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout := `proposed_memory:
  - type: workflow
    title: Stdout YAML proposals are rejected
    trigger: When worker stdout is YAML.
    summary: SIMA requires JSON worker stdout.
`
	if err := os.WriteFile(filepath.Join(runDir, "stdout.log"), []byte(stdout), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{FromRun: runID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Source != "structured_invalid" || result.Candidates != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"candidate_source: structured_invalid", "worker stdout must be JSON"} {
		if !strings.Contains(text, want) {
			t.Fatalf("proposal missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateUsesStructuredJSONFromStdout(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	runID := "20260824-170000-json"
	runDir := filepath.Join(root, ".sima", "personal", "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "worker-report.yaml"), []byte("run_id: "+runID+"\nstatus: success\nexit_code: 0\ntask: json stdout proposals\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout := `{"proposed_memory":[{"type":"workflow","title":"JSON proposals are parsed","trigger":"When worker stdout is JSON.","summary":"SIMA can parse JSON worker proposals without YAML scalar pitfalls."}]}`
	if err := os.WriteFile(filepath.Join(runDir, "stdout.log"), []byte(stdout), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{FromRun: runID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Source != "structured" || result.Candidates != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"JSON proposals are parsed", "destination: memory"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("proposal missing %q:\n%s", want, data)
		}
	}
}

func TestGenerateUsesClaudeStructuredOutputWrapper(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	runID := "20260824-171000-wrapper"
	runDir := filepath.Join(root, ".sima", "personal", "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "worker-report.yaml"), []byte("run_id: "+runID+"\nstatus: success\nexit_code: 0\ntask: json schema wrapper\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout := `{"type":"result","subtype":"success","result":"prose wrapper","structured_output":{"proposed_memory":[{"type":"workflow","title":"Claude schema wrapper is parsed","trigger":"When Claude Code returns --output-format json with structured_output.","summary":"SIMA extracts structured_output and ignores the prose wrapper result."}]}}`
	if err := os.WriteFile(filepath.Join(runDir, "stdout.log"), []byte(stdout), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{FromRun: runID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Source != "structured" || result.Candidates != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "Claude schema wrapper is parsed") {
		t.Fatalf("proposal did not use structured_output candidate:\n%s", data)
	}
}

func TestGenerateDoesNotFallbackOnMalformedStructuredStdout(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	runID := "20260824-160000-malformed"
	runDir := filepath.Join(root, ".sima", "personal", "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "worker-report.yaml"), []byte("run_id: "+runID+"\nstatus: success\nexit_code: 0\ntask: malformed stdout proposals\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout := "proposed_memory:\n  - type: workflow\n    title: Missing quote: \"broken\n"
	if err := os.WriteFile(filepath.Join(runDir, "stdout.log"), []byte(stdout), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{FromRun: runID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Source != "structured_invalid" || result.Candidates != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"candidate_source: structured_invalid", "candidate_errors:", "worker stdout must be JSON"} {
		if !strings.Contains(text, want) {
			t.Fatalf("proposal missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Review successful SIMA run for durable lessons") {
		t.Fatalf("malformed structured stdout should not fall back:\n%s", text)
	}
}

func TestGenerateMarksIncompleteStructuredCandidatesInvalid(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	runID := "20260824-161000-incomplete"
	runDir := filepath.Join(root, ".sima", "personal", "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	report := `run_id: 20260824-161000-incomplete
status: success
exit_code: 0
task: incomplete structured proposals
proposed_memory:
  - type: gotcha
    title: Missing trigger and summary
`
	if err := os.WriteFile(filepath.Join(runDir, "worker-report.yaml"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{FromRun: runID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Source != "structured_invalid" || result.Candidates != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"candidate_source: structured_invalid", "proposed_memory[0] requires type, title, trigger, and summary"} {
		if !strings.Contains(text, want) {
			t.Fatalf("proposal missing %q:\n%s", want, text)
		}
	}
}
