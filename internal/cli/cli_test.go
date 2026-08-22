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
	if !strings.Contains(out.String(), "Exit code: 0") {
		t.Fatalf("unexpected run output: %q", out.String())
	}
	entries, err := os.ReadDir(filepath.Join(root, ".sima", "personal", "runs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one run, got %d", len(entries))
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
