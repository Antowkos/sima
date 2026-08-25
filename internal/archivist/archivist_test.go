package archivist

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
	if !strings.Contains(string(data), "archivist_decision: reject") || !strings.Contains(string(data), "status: rejected") {
		t.Fatalf("proposal not rejected:\n%s", data)
	}
}

func TestDecideDefersNoStructuredLearningCandidates(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := config.AddBackend(root, "worker", config.BackendProfile{Kind: "codex", Executable: "/bin/echo"}, false); err != nil {
		t.Fatalf("AddBackend() error = %v", err)
	}
	runResult, err := runner.Run(root, runner.Options{BackendName: "worker", Task: "no learning proposal", Now: time.Date(2026, 8, 23, 12, 30, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	proposalResult, err := proposal.Generate(root, proposal.Options{FromRun: runResult.RunID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	result, err := Decide(root, Options{Target: proposalResult.Path})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if result.Decision != "defer" || !strings.Contains(strings.Join(result.Notes, "\n"), "no structured learning candidates") {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(proposalResult.Path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "status: session_only") || strings.Contains(text, "candidate_source: fallback") {
		t.Fatalf("proposal not deferred without fallback:\n%s", data)
	}
}

func TestDecideDefersFailedLearningQuality(t *testing.T) {
	root, proposalPath := createProposal(t, "archive low quality proposal", false)
	data, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "durable: true", "durable: false", 1)
	if err := os.WriteFile(proposalPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Decide(root, Options{Target: proposalPath})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if result.Decision != "defer" || !strings.Contains(strings.Join(result.Notes, "\n"), "durable=false") {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err = os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "status: deferred") {
		t.Fatalf("proposal not deferred:\n%s", data)
	}
}

func TestDecideDefersSimilarActiveMemory(t *testing.T) {
	root, proposalPath := createProposal(t, "archive duplicate proposal", false)
	activeDir := filepath.Join(root, ".sima", "personal", "memory", "cards")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	active := `id: existing
status: active
type: workflow
title: Structured archivist approval
trigger: When a worker emits structured proposed_memory.
summary: Existing active card with the same durable lesson.
`
	if err := os.WriteFile(filepath.Join(activeDir, "existing.yaml"), []byte(active), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Decide(root, Options{Target: proposalPath})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	joined := strings.Join(result.Notes, "\n")
	if result.Decision != "defer" || !strings.Contains(joined, "similar active memory") {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "status: deferred") || !strings.Contains(string(data), "manual dedup/update needed") {
		t.Fatalf("proposal not deferred for dedup:\n%s", data)
	}
}

func TestDecideDefersSimilarActiveSkill(t *testing.T) {
	root, proposalPath := createSkillProposal(t)
	activeDir := filepath.Join(root, ".sima", "personal", "skills", "active")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	active := `---
name: structured-proposal-skill
---
# structured-proposal-skill

## Trigger

When a worker emits structured proposed_skills.
`
	if err := os.WriteFile(filepath.Join(activeDir, "structured-proposal-skill.md"), []byte(active), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Decide(root, Options{Target: proposalPath})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	joined := strings.Join(result.Notes, "\n")
	if result.Decision != "defer" || !strings.Contains(joined, "similar active skill") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestDecideUsesModelArchivistBackend(t *testing.T) {
	root, proposalPath := createProposal(t, "model archivist proposal", false)
	script := filepath.Join(root, "model-archivist.sh")
	output := `{"decision":"apply","learning":{"destination":"memory","operation":"create","quality":{"durable":true,"triggerable":true,"evidence_backed":true,"non_transient":true,"reusable":true},"notes":["model quality passed"]},"notes":["model approved bounded evidence"]}`
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' '"+output+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := config.AddBackend(root, "archivist-model", config.BackendProfile{Kind: "claude-code", Executable: script}, false); err != nil {
		t.Fatalf("AddBackend() error = %v", err)
	}

	result, err := Decide(root, Options{Target: proposalPath, BackendName: "archivist-model", Now: time.Date(2026, 8, 23, 14, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	joined := strings.Join(result.Notes, "\n")
	if result.Decision != "apply" || !strings.Contains(joined, "model archivist reviewed") || !strings.Contains(joined, "model approved bounded evidence") {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "archivist_decision: apply") || !strings.Contains(text, "model quality passed") {
		t.Fatalf("proposal not updated with model decision:\n%s", text)
	}
}

func TestDecideAllowsModelRejectWithoutLifecycleTarget(t *testing.T) {
	root, proposalPath := createProposal(t, "model archivist reject proposal", false)
	script := filepath.Join(root, "model-archivist-reject.sh")
	output := `{"decision":"reject","learning":{"destination":"reject","operation":"deprecate","quality":{"durable":false,"triggerable":false,"evidence_backed":false,"non_transient":false,"reusable":false},"notes":["not evidence-backed"]},"notes":["model rejected fabricated learning"]}`
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' '"+output+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := config.AddBackend(root, "archivist-reject", config.BackendProfile{Kind: "claude-code", Executable: script}, false); err != nil {
		t.Fatalf("AddBackend() error = %v", err)
	}

	result, err := Decide(root, Options{Target: proposalPath, BackendName: "archivist-reject"})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	joined := strings.Join(result.Notes, "\n")
	if result.Decision != "reject" || strings.Contains(joined, "target.path is required") || !strings.Contains(joined, "model rejected fabricated learning") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestBuildArchivistPromptIncludesFullEvidenceAndActiveKnowledge(t *testing.T) {
	root, proposalPath := createProposal(t, "prompt packet proposal", false)
	stdoutPath := filepath.Join(root, ".sima", "personal", "runs", "20260823-123000-worker", "stdout.log")
	if err := os.WriteFile(stdoutPath, []byte("first evidence line\nsecond evidence line with unicode: Привет мир\nthird evidence line should not be truncated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	activeDir := filepath.Join(root, ".sima", "personal", "memory", "cards")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(activeDir, "active.yaml"), []byte("id: active\nstatus: active\ntitle: Active packet memory\nsummary: visible active knowledge\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(activeDir, "deprecated.yaml"), []byte("id: old\nstatus: deprecated\ntitle: Deprecated packet memory\nsummary: hidden inactive knowledge\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	prompt, err := buildArchivistPrompt(root, proposalPath)
	if err != nil {
		t.Fatalf("buildArchivistPrompt() error = %v", err)
	}
	for _, want := range []string{
		"---BEGIN EVIDENCE PACKET---",
		"second evidence line with unicode: Привет мир",
		"third evidence line should not be truncated",
		"---BEGIN ACTIVE KNOWLEDGE---",
		"Active packet memory",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "Deprecated packet memory") {
		t.Fatalf("prompt included inactive knowledge:\n%s", prompt)
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
	} else {
		reportPath := root + "/.sima/personal/runs/" + runResult.RunID + "/worker-report.yaml"
		report := "run_id: " + runResult.RunID + "\nstatus: success\nexit_code: 0\ntask: " + task + "\nproposed_memory:\n  - type: workflow\n    title: Structured archivist approval\n    trigger: When a worker emits structured proposed_memory.\n    summary: The deterministic archivist may auto-apply safe structured personal proposals.\n"
		if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	proposalResult, err := proposal.Generate(root, proposal.Options{FromRun: runResult.RunID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	return root, proposalResult.Path
}

func createSkillProposal(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := config.AddBackend(root, "worker", config.BackendProfile{Kind: "codex", Executable: "/bin/echo"}, false); err != nil {
		t.Fatalf("AddBackend() error = %v", err)
	}
	runResult, err := runner.Run(root, runner.Options{BackendName: "worker", Task: "skill proposal", Now: time.Date(2026, 8, 23, 12, 30, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	reportPath := root + "/.sima/personal/runs/" + runResult.RunID + "/worker-report.yaml"
	report := "run_id: " + runResult.RunID + "\nstatus: success\nexit_code: 0\ntask: skill proposal\nproposed_skills:\n  - name: structured-proposal-skill\n    trigger: When a worker emits structured proposed_skills.\n    summary: Convert worker-proposed skills into reusable active skill files with evidence.\n"
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	proposalResult, err := proposal.Generate(root, proposal.Options{FromRun: runResult.RunID})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	return root, proposalResult.Path
}
