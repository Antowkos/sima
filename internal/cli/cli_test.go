package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antowkos/sima/internal/simafs"
)

func TestVersion(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"sima", "version"}, &out, &err)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, err.String())
	}
	if !strings.Contains(out.String(), "sima ") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"sima", "wat"}, &out, &err)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %s", code, err.String())
	}
	if !strings.Contains(err.String(), "unknown command") {
		t.Fatalf("unexpected stderr: %q", err.String())
	}
}

func TestBriefCommand(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "brief", "implement", "backend", "run", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("brief code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "Brief written:") {
		t.Fatalf("unexpected brief output: %q", out.String())
	}
	entries, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "briefs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one brief, got %d", len(entries))
	}
}

func TestRunCommand(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "echo", "--kind", "codex", "--executable", "/bin/echo", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "run", "--backend", "echo", "--task", "capture artifacts", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "Exit code: 0") || !strings.Contains(out.String(), "Proposal written:") || !strings.Contains(out.String(), "Next: sima review") {
		t.Fatalf("unexpected run output: %q", out.String())
	}
	entries, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "runs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one run, got %d", len(entries))
	}
	proposalEntries, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "candidates"))
	if err != nil {
		t.Fatal(err)
	}
	if len(proposalEntries) != 1 {
		t.Fatalf("expected one auto proposal, got %d", len(proposalEntries))
	}
}

func TestRunCommandNoPropose(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "echo", "--kind", "codex", "--executable", "/bin/echo", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "run", "--backend", "echo", "--task", "capture artifacts", "--path", root, "--no-propose"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if strings.Contains(out.String(), "Proposal written:") {
		t.Fatalf("unexpected auto proposal output: %q", out.String())
	}
	proposalEntries, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "candidates"))
	if err != nil {
		t.Fatal(err)
	}
	if len(proposalEntries) != 0 {
		t.Fatalf("expected no auto proposal, got %d", len(proposalEntries))
	}
}

func TestProposeCommand(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "echo", "--kind", "codex", "--executable", "/bin/echo", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "run", "--backend", "echo", "--task", "capture artifacts", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "propose", "--from-run", "last", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("propose code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "Proposal written:") || !strings.Contains(out.String(), "Archivist decision: defer") {
		t.Fatalf("unexpected propose output: %q", out.String())
	}
	entries, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "candidates"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one proposal, got %d", len(entries))
	}
}

func TestReviewCommand(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "echo", "--kind", "codex", "--executable", "/bin/echo", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "run", "--backend", "echo", "--task", "capture artifacts", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "propose", "--from-run", "last", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("propose code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "review", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("review code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "valid") || !strings.Contains(out.String(), "destination=session_only") || !strings.Contains(out.String(), "operation=create") || !strings.Contains(out.String(), "Summary: 1 total, 1 valid, 0 blocked") {
		t.Fatalf("unexpected review output: %q", out.String())
	}
}

func TestApplyCommand(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "echo", "--kind", "codex", "--executable", "/bin/echo", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "run", "--backend", "echo", "--task", "capture artifacts", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	runEntries, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "runs"))
	if err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, ".sima", "personal", "runs", runEntries[0].Name(), "worker-report.yaml")
	report := "run_id: " + runEntries[0].Name() + "\nstatus: success\nexit_code: 0\ntask: capture artifacts\nproposed_memory:\n  - type: workflow\n    title: CLI apply approved structured proposal\n    trigger: When the SIMA CLI applies an archivist-approved structured proposal.\n    summary: The apply command promotes structured personal proposal candidates after archivist approval.\n"
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "propose", "--from-run", "last", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("propose code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	entries, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "candidates"))
	if err != nil {
		t.Fatal(err)
	}
	proposalPath := filepath.Join(root, ".sima", "personal", "memory", "candidates", entries[0].Name())
	data, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proposalPath, []byte(strings.Replace(string(data), "archivist_decision: defer", "archivist_decision: apply", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "apply", proposalPath, "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("apply code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "Applied proposal:") || !strings.Contains(out.String(), ".sima/personal/memory/cards/") {
		t.Fatalf("unexpected apply output: %q", out.String())
	}
}

func TestArchivistCommand(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "echo", "--kind", "codex", "--executable", "/bin/echo", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "run", "--backend", "echo", "--task", "capture artifacts", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "propose", "--from-run", "last", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("propose code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	entries, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "candidates"))
	if err != nil {
		t.Fatal(err)
	}
	proposalID := strings.TrimSuffix(entries[0].Name(), ".yaml")
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "archivist", "--proposal", proposalID, "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("archivist code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "Archivist decision: defer") || !strings.Contains(out.String(), "proposal has no structured learning candidates") {
		t.Fatalf("unexpected archivist output: %q", out.String())
	}
}

func TestLearnCommandStopsWhenWorkerProposesNoLearning(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "echo", "--kind", "codex", "--executable", "/bin/echo", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "learn", "--backend", "echo", "--task", "capture and apply safe lesson", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("learn code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	for _, want := range []string{"Run ", "Proposal written:", "Candidates: 0", "Learn stopped: no structured learning candidates or lifecycle operation; no fallback proposal or archivist review attempted"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("learn output missing %q: %q", want, out.String())
		}
	}
	if strings.Contains(out.String(), "Archivist decision:") {
		t.Fatalf("learn should not run archivist for no-candidate worker output: %q", out.String())
	}
	cards, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "cards"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 0 {
		t.Fatalf("expected no applied cards for no-candidate worker output, got %d", len(cards))
	}

	proposals, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "candidates"))
	if err != nil {
		t.Fatal(err)
	}
	if len(proposals) != 1 {
		t.Fatalf("expected one audit proposal, got %d", len(proposals))
	}
	data, err := os.ReadFile(filepath.Join(root, ".sima", "personal", "memory", "candidates", proposals[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "status: candidate") || strings.Contains(text, "candidate_source: fallback") || strings.Contains(text, "Review successful SIMA run for durable lessons") {
		t.Fatalf("proposal should not contain fallback candidate:\n%s", data)
	}
}

func TestLearnCommandAppliesStructuredCandidate(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	worker := filepath.Join(root, "structured-worker.sh")
	if err := os.WriteFile(worker, []byte("#!/bin/sh\ncat <<'JSON'\n{\"proposed_memory\":[{\"type\":\"workflow\",\"title\":\"Structured learn candidate\",\"trigger\":\"When sima learn receives structured worker proposals.\",\"summary\":\"sima learn may auto-apply safe structured personal proposals.\"}]}\nJSON\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "structured", "--kind", "codex", "--executable", worker, "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}
	reviewer := filepath.Join(root, "reviewer.sh")
	reviewerJSON := `{"decision":"apply","learning":{"destination":"memory","operation":"create","quality":{"durable":true,"triggerable":true,"evidence_backed":true,"non_transient":true,"reusable":true},"notes":["model reviewer approved structured learning"]},"notes":["clean reviewer approved"]}`
	if err := os.WriteFile(reviewer, []byte("#!/bin/sh\nprintf '%s\\n' '"+reviewerJSON+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "backend", "add", "reviewer", "--kind", "claude-code", "--executable", reviewer, "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("reviewer backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "learn", "--backend", "structured", "--archivist-backend", "reviewer", "--task", "capture and apply structured lesson", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("learn code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	for _, want := range []string{"Run ", "Proposal written:", "Archivist backend: reviewer", "Archivist decision: apply", "Applied proposal:", "Learn complete:"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("learn output missing %q: %q", want, out.String())
		}
	}
	cards, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "cards"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected one applied card, got %d", len(cards))
	}
	proposals, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "candidates"))
	if err != nil {
		t.Fatal(err)
	}
	if len(proposals) != 1 {
		t.Fatalf("expected one proposal, got %d", len(proposals))
	}
	data, err := os.ReadFile(filepath.Join(root, ".sima", "personal", "memory", "candidates", proposals[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "status: applied") || !strings.Contains(string(data), "archivist_decision: apply") || !strings.Contains(string(data), "candidate_source: structured") {
		t.Fatalf("proposal not marked applied:\n%s", data)
	}
}

func TestLintCommand(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	memoryDir := filepath.Join(root, ".sima", "personal", "memory", "cards")
	if err := os.WriteFile(filepath.Join(memoryDir, "bad.yaml"), []byte("id: bad\nstatus: stale\ntitle: Bad status\nsummary: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "lint", root}, &out, &stderr)
	if code != 1 {
		t.Fatalf("lint code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "SIMA lint for") || !strings.Contains(out.String(), "status must be active") || !strings.Contains(out.String(), "Summary: 1 errors") {
		t.Fatalf("unexpected lint output: %q", out.String())
	}
}

func TestCandidatesListAndShowCommands(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	candidateDir := filepath.Join(root, ".sima", "personal", "memory", "candidates")
	candidate := "version: 1\nid: inspect-me\nkind: run_reflection\nscope: personal\noperation: create\nstatus: candidate\narchivist_decision: apply\nsafety:\n  decision: safe\nlearning:\n  destination: memory\n  operation: create\ncandidate_memories:\n  - type: invariant\n    title: Inspect candidates\n    trigger: When candidate queues need review.\n    summary: Candidate list and show expose proposal metadata before mutation.\nrun:\n  id: run\n  path: .sima/personal/runs/run\n"
	if err := os.WriteFile(filepath.Join(candidateDir, "inspect-me.yaml"), []byte(candidate), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "candidates", "list", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("candidates list code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	for _, want := range []string{"STATUS\tDECISION\tSAFETY\tDESTINATION\tOPERATION\tCANDIDATES\tID\tPATH", "candidate\tapply\tsafe\tmemory\tcreate\t1\tinspect-me\t.sima/personal/memory/candidates/inspect-me.yaml"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("candidate list missing %q:\n%s", want, out.String())
		}
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "candidates", "show", "inspect-me", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("candidates show code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	for _, want := range []string{"Candidate: .sima/personal/memory/candidates/inspect-me.yaml", "ID: inspect-me", "Status: candidate", "title: Inspect candidates"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("candidate show missing %q:\n%s", want, out.String())
		}
	}
}

func TestCandidatesCleanupCommand(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	candidateDir := filepath.Join(root, ".sima", "personal", "memory", "candidates")
	candidate := "version: 1\nid: deferred\nkind: run_reflection\nscope: personal\noperation: create\nstatus: candidate\narchivist_decision: defer\nsafety:\n  decision: safe\nrun:\n  id: run\n  path: .sima/personal/runs/run\n"
	if err := os.WriteFile(filepath.Join(candidateDir, "deferred.yaml"), []byte(candidate), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "candidates", "cleanup", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("candidates cleanup code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "Cleaned candidates: 1") || !strings.Contains(out.String(), ".sima/personal/memory/candidates/deferred.yaml") {
		t.Fatalf("unexpected cleanup output: %q", out.String())
	}
	data, err := os.ReadFile(filepath.Join(candidateDir, "deferred.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "status: deferred") {
		t.Fatalf("candidate not marked deferred:\n%s", data)
	}
}

func TestMemoryAndSkillListCommandsShowLifecycleStatus(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	memoryDir := filepath.Join(root, ".sima", "personal", "memory", "cards")
	if err := os.WriteFile(filepath.Join(memoryDir, "active.yaml"), []byte("id: active\nstatus: active\ntitle: Active Memory\nsummary: visible\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memoryDir, "deprecated.yaml"), []byte("id: old\nstatus: deprecated\ntitle: Deprecated Memory\nsummary: hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(root, ".sima", "personal", "skills", "active")
	if err := os.WriteFile(filepath.Join(skillDir, "active-skill.md"), []byte("---\nname: active-skill\nstatus: active\n---\n# Active Skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "superseded-skill.md"), []byte("---\nname: superseded-skill\nstatus: superseded\n---\n# Superseded Skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "memory", "list", "--status", "all", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("memory list code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	for _, want := range []string{"STATUS\tSCOPE\tKIND\tTITLE\tPATH", "active\tpersonal\tmemory\tActive Memory\t.sima/personal/memory/cards/active.yaml", "deprecated\tpersonal\tmemory\tDeprecated Memory\t.sima/personal/memory/cards/deprecated.yaml"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("memory list missing %q:\n%s", want, out.String())
		}
	}

	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "skill", "list", "--status", "active", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("skill list code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "active\tpersonal\tskill\tactive-skill\t.sima/personal/skills/active/active-skill.md") || strings.Contains(out.String(), "superseded-skill") {
		t.Fatalf("unexpected active skill list:\n%s", out.String())
	}
}

func TestBackendAddListDoctor(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}

	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "test", "--kind", "codex", "--executable", "/bin/echo", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}

	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "backend", "list", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend list code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(out.String(), "test\tcodex\t/bin/echo") {
		t.Fatalf("unexpected list output: %q", out.String())
	}

	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "backend", "doctor", "test", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend doctor code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "[ok] test") {
		t.Fatalf("unexpected doctor output: %q", out.String())
	}

	if _, statErr := os.Stat(filepath.Join(root, ".sima", "config.yaml")); statErr != nil {
		t.Fatalf("config.yaml missing: %v", statErr)
	}
}
