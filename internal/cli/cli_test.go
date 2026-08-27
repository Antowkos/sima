package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antowkos/sima/internal/config"
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

func TestInstallCommand(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "install", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("install code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "CLAUDE.md") || !strings.Contains(out.String(), "AGENTS.md") {
		t.Fatalf("unexpected install output: %q", out.String())
	}
	for _, rel := range []string{"CLAUDE.md", "AGENTS.md"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if !strings.Contains(string(data), "BEGIN SIMA MANAGED INSTRUCTIONS") || !strings.Contains(string(data), "sima learn --backend <backend-name>") {
			t.Fatalf("missing managed instructions in %s:\n%s", rel, data)
		}
		if !strings.Contains(string(data), "route it through the SIMA harness") || !strings.Contains(string(data), "sima remember") || !strings.Contains(string(data), "native memory") {
			t.Fatalf("missing explicit SIMA remember instructions in %s:\n%s", rel, data)
		}
		if !strings.Contains(string(data), "SIMA-managed PR fixes") || !strings.Contains(string(data), "Address PR review comments using gh/repo inspection") {
			t.Fatalf("missing SIMA-managed PR fix instructions in %s:\n%s", rel, data)
		}
	}
}

func TestRememberCommandCreatesSimaHarnessCandidate(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	var out, stderr bytes.Buffer
	knowledge := "Use WidgetKit disabled empty state instead of demo data in production widgets."
	trigger := "When implementing production WidgetKit fallback behavior."
	code := Run([]string{"sima", "remember", knowledge, "--path", root, "--source", "user", "--type", "guardrail", "--title", "Production widgets avoid demo data", "--trigger", trigger}, &out, &stderr)
	if code != 0 {
		t.Fatalf("remember code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "Remember proposal written") || !strings.Contains(out.String(), "Next: sima archivist") {
		t.Fatalf("unexpected remember output: %q", out.String())
	}
	entries, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "candidates"))
	if err != nil {
		t.Fatalf("read candidates: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one candidate, got %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(root, ".sima", "personal", "memory", "candidates", entries[0].Name()))
	if err != nil {
		t.Fatalf("read candidate: %v", err)
	}
	text := string(data)
	for _, want := range []string{"kind: direct_knowledge", "candidate_source: structured", "type: guardrail", "title: Production widgets avoid demo data", "explicit knowledge routed through SIMA harness"} {
		if !strings.Contains(text, want) {
			t.Fatalf("candidate missing %q:\n%s", want, text)
		}
	}
}

func TestSetupCommandBackendNone(t *testing.T) {
	root := t.TempDir()
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "setup", "--path", root, "--backend", "none"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("setup code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "Skipped backend setup") || !strings.Contains(out.String(), "Running lint preflight") {
		t.Fatalf("unexpected setup output: %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".sima", "config.yaml")); err != nil {
		t.Fatalf("missing config: %v", err)
	}
	for _, rel := range []string{"CLAUDE.md", "AGENTS.md"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if !strings.Contains(string(data), "BEGIN SIMA MANAGED INSTRUCTIONS") {
			t.Fatalf("missing managed instructions in %s", rel)
		}
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.Backends) != 0 {
		t.Fatalf("expected no backends, got %#v", cfg.Backends)
	}
}

func TestSetupCommandAcceptsPositionalPath(t *testing.T) {
	root := t.TempDir()
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "setup", root, "--backend", "none"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("setup code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".sima", "config.yaml")); err != nil {
		t.Fatalf("missing config: %v", err)
	}
}

func TestSetupCommandDefaultsToCurrentDirectory(t *testing.T) {
	root := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "setup", "--backend", "none"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("setup code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	expectedRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs cwd: %v", err)
	}
	if !strings.Contains(out.String(), "Setting up SIMA project in "+expectedRoot) {
		t.Fatalf("setup did not use current directory: %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".sima", "config.yaml")); err != nil {
		t.Fatalf("missing config in cwd: %v", err)
	}
}

func TestSetupCommandAutoAddsClaudeBackend(t *testing.T) {
	root := t.TempDir()
	bin := t.TempDir()
	fakeClaude := filepath.Join(bin, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	t.Setenv("PATH", bin)

	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "setup", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("setup code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "Added backend: claude-main") || !strings.Contains(out.String(), "SIMA doctor") {
		t.Fatalf("unexpected setup output: %q", out.String())
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	profile, ok := cfg.Backends["claude-main"]
	if !ok {
		t.Fatalf("claude-main backend missing: %#v", cfg.Backends)
	}
	if profile.Kind != "claude-code" || profile.Executable != fakeClaude {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}

func TestSetupCommandUsesExplicitCodexExecutable(t *testing.T) {
	root := t.TempDir()
	bin := t.TempDir()
	fakeCodex := filepath.Join(bin, "custom-codex")
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv("PATH", "/usr/bin:/bin")

	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "setup", "--path", root, "--backend", "codex", "--executable", fakeCodex}, &out, &stderr)
	if code != 0 {
		t.Fatalf("setup code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "Added backend: codex-main") {
		t.Fatalf("unexpected setup output: %q", out.String())
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	profile, ok := cfg.Backends["codex-main"]
	if !ok {
		t.Fatalf("codex-main backend missing: %#v", cfg.Backends)
	}
	if profile.Kind != "codex" || profile.Executable != fakeCodex || profile.PermissionMode != "workspace-write" {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}

func TestSetupCommandAutoUsesExplicitClaudeExecutable(t *testing.T) {
	root := t.TempDir()
	bin := t.TempDir()
	fakeClaude := filepath.Join(bin, "custom-claude")
	configDir := filepath.Join(root, "claude-config")
	if err := os.Mkdir(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	t.Setenv("PATH", "/usr/bin:/bin")

	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "setup", "--path", root, "--claude-executable", fakeClaude, "--claude-config-dir", configDir, "--env", "EXTRA_FLAG=1"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("setup code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	profile, ok := cfg.Backends["claude-main"]
	if !ok {
		t.Fatalf("claude-main backend missing: %#v", cfg.Backends)
	}
	if profile.Kind != "claude-code" || profile.Executable != fakeClaude {
		t.Fatalf("unexpected profile: %#v", profile)
	}
	if profile.Env["CLAUDE_CONFIG_DIR"] != configDir || profile.Env["EXTRA_FLAG"] != "1" {
		t.Fatalf("unexpected env: %#v", profile.Env)
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
	code = Run([]string{"sima", "learn", "--backend", "echo", "--task", "capture and apply safe lesson", "--no-auto-cleanup-deferred", "--path", root}, &out, &stderr)
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

func TestLearnCommandAutoCleanupDeferred(t *testing.T) {
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
	code = Run([]string{"sima", "learn", "--backend", "echo", "--task", "capture no durable lesson", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("learn auto cleanup code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	for _, want := range []string{"Learn stopped: no structured learning candidates or lifecycle operation", "Deferred cleanup: 1"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("learn auto cleanup output missing %q: %q", want, out.String())
		}
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
	if !strings.Contains(string(data), "status: deferred") || !strings.Contains(string(data), "cleanup_note:") {
		t.Fatalf("proposal should be cleaned to deferred:\n%s", data)
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
	for _, want := range []string{"Run ", "Proposal written:", "Archivist backend: reviewer", "Archivist decision: apply", "Learn auto-apply: proposal passed apply-ready gates", "Applied proposal:", "Learn complete:"} {
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

func TestLearnCommandJSONSummary(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	worker := filepath.Join(root, "structured-worker.sh")
	if err := os.WriteFile(worker, []byte("#!/bin/sh\ncat <<'JSON'\n{\"proposed_memory\":[{\"type\":\"workflow\",\"title\":\"Structured JSON learn candidate\",\"trigger\":\"When sima learn --json runs.\",\"summary\":\"sima learn --json should emit a machine-readable final summary.\"}]}\nJSON\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	reviewer := filepath.Join(root, "reviewer.sh")
	reviewerJSON := `{"decision":"apply","learning":{"destination":"memory","operation":"create","quality":{"durable":true,"triggerable":true,"evidence_backed":true,"non_transient":true,"reusable":true},"notes":["model reviewer approved structured learning"]},"notes":["clean reviewer approved"]}`
	if err := os.WriteFile(reviewer, []byte("#!/bin/sh\nprintf '%s\\n' '"+reviewerJSON+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "structured", "--kind", "codex", "--executable", worker, "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "backend", "add", "reviewer", "--kind", "claude-code", "--executable", reviewer, "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("reviewer backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "learn", "--backend", "structured", "--archivist-backend", "reviewer", "--task", "capture json structured lesson", "--json", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("learn --json code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if strings.Contains(out.String(), "Run ") || strings.Contains(out.String(), "Learn complete:") {
		t.Fatalf("json output should not include human log lines: %s", out.String())
	}
	var summary struct {
		Status            string   `json:"status"`
		Outcome           string   `json:"outcome"`
		Candidates        int      `json:"candidates"`
		ApplyReady        bool     `json:"apply_ready"`
		AppliedPaths      []string `json:"applied_paths"`
		ArchivistDecision string   `json:"archivist_decision"`
	}
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil {
		t.Fatalf("json summary parse failed: %v\n%s", err, out.String())
	}
	if summary.Status != "completed" || summary.Outcome != "applied" || summary.Candidates != 1 || !summary.ApplyReady || len(summary.AppliedPaths) != 1 || summary.ArchivistDecision != "apply" {
		t.Fatalf("unexpected json summary: %#v\n%s", summary, out.String())
	}
}

func TestLearnCommandConfigNoAutoApplyLeavesReadyProposalPending(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Learn.AutoApply = false
	cfg.Learn.AutoCleanupDeferred = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	worker := filepath.Join(root, "structured-worker.sh")
	if err := os.WriteFile(worker, []byte("#!/bin/sh\ncat <<'JSON'\n{\"proposed_memory\":[{\"type\":\"workflow\",\"title\":\"Structured inspect-only learn candidate\",\"trigger\":\"When sima learn runs with no-auto-apply.\",\"summary\":\"sima learn should leave apply-ready proposals pending when --no-auto-apply is set.\"}]}\nJSON\n"), 0o755); err != nil {
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
	code = Run([]string{"sima", "learn", "--backend", "structured", "--archivist-backend", "reviewer", "--task", "capture inspect-only structured lesson", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("learn config auto_apply false code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	for _, want := range []string{"Archivist decision: apply", "Learn auto-apply: proposal passed apply-ready gates", "Learn stopped: auto_apply disabled; proposal remains pending"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("learn --no-auto-apply output missing %q: %q", want, out.String())
		}
	}
	if strings.Contains(out.String(), "Applied proposal:") {
		t.Fatalf("learn --no-auto-apply should not apply: %q", out.String())
	}
	cards, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "memory", "cards"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 0 {
		t.Fatalf("expected no applied cards, got %d", len(cards))
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
	if !strings.Contains(string(data), "status: candidate") || !strings.Contains(string(data), "archivist_decision: apply") {
		t.Fatalf("proposal should remain pending/apply-ready:\n%s", data)
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
	code = Run([]string{"sima", "candidates", "apply-ready", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("candidates apply-ready code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	if strings.Contains(out.String(), "inspect-me") {
		t.Fatalf("candidate without quality flags should not be apply-ready:\n%s", out.String())
	}
	for _, want := range []string{"STATUS	DECISION	SAFETY	DESTINATION	OPERATION	CANDIDATES	ID	PATH"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("candidate apply-ready missing %q:\n%s", want, out.String())
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

func TestCandidatesApplyReadyApplyCommand(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	candidateDir := filepath.Join(root, ".sima", "personal", "memory", "candidates")
	candidate := `version: 1
id: ready
kind: run_reflection
scope: personal
operation: create
status: candidate
archivist_decision: apply
safety:
  decision: safe
learning:
  destination: memory
  operation: create
  quality:
    durable: true
    triggerable: true
    evidence_backed: true
    non_transient: true
    reusable: true
candidate_memories:
  - type: invariant
    title: Bulk apply ready candidates
    trigger: When applying multiple reviewed SIMA proposals.
    summary: Bulk apply should mutate only proposals that already pass the apply-ready gates.
    evidence:
      - kind: task
        path: .sima/personal/runs/run/task.md
        note: original task
run:
  id: run
  path: .sima/personal/runs/run
evidence:
  - kind: task
    path: .sima/personal/runs/run/task.md
    note: original task
`
	proposalPath := filepath.Join(candidateDir, "ready.yaml")
	if err := os.WriteFile(proposalPath, []byte(candidate), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "candidates", "apply-ready", "--apply", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("candidates apply-ready --apply code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	for _, want := range []string{"Applying candidates: 1", ".sima/personal/memory/candidates/ready.yaml", "applied: .sima/personal/memory/cards/ready-01-bulk-apply-ready-candidates.yaml"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("apply-ready --apply missing %q:\n%s", want, out.String())
		}
	}
	proposalData, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(proposalData), "status: applied") {
		t.Fatalf("proposal not marked applied:\n%s", proposalData)
	}
	cardData, err := os.ReadFile(filepath.Join(root, ".sima", "personal", "memory", "cards", "ready-01-bulk-apply-ready-candidates.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cardData), "status: active") || !strings.Contains(string(cardData), "title: Bulk apply ready candidates") {
		t.Fatalf("memory card not created correctly:\n%s", cardData)
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

func TestDoctorRequiresBackendForAlphaPreflight(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}

	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "doctor", root}, &out, &stderr)
	if code == 0 {
		t.Fatalf("doctor unexpectedly passed without backend: stdout = %s", out.String())
	}
	if !strings.Contains(out.String(), "[fail] backends: configured") || !strings.Contains(stderr.String(), "SIMA doctor found problems") {
		t.Fatalf("doctor missing backend failure: stdout = %s stderr = %s", out.String(), stderr.String())
	}
}

func TestDoctorPassesAlphaPreflightWithBackend(t *testing.T) {
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
	code = Run([]string{"sima", "doctor", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("doctor code = %d, stdout = %s stderr = %s", code, out.String(), stderr.String())
	}
	for _, want := range []string{"[ok] config: learn auto_apply", "[ok] config: learn auto_cleanup_deferred", "[ok] backends: configured", "[ok] backend: test", "[ok] lint: errors: 0 errors, 0 warnings", "[ok] candidates: queue: 0 pending candidates"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("doctor output missing %q:\n%s", want, out.String())
		}
	}
}

func TestDoctorFailsWhenAutoLearningDisabled(t *testing.T) {
	root := t.TempDir()
	if _, initErr := simafs.Init(root); initErr != nil {
		t.Fatalf("Init() error = %v", initErr)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Learn.AutoApply = false
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}

	var out, stderr bytes.Buffer
	code := Run([]string{"sima", "backend", "add", "test", "--kind", "codex", "--executable", "/bin/echo", "--path", root}, &out, &stderr)
	if code != 0 {
		t.Fatalf("backend add code = %d, stderr = %s", code, stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run([]string{"sima", "doctor", root}, &out, &stderr)
	if code == 0 {
		t.Fatalf("doctor unexpectedly passed with auto_apply disabled: stdout = %s", out.String())
	}
	if !strings.Contains(out.String(), "[fail] config: learn auto_apply") {
		t.Fatalf("doctor missing auto_apply failure: stdout = %s", out.String())
	}
}
