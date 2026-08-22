package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddBackendRoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".sima"), 0o755); err != nil {
		t.Fatal(err)
	}
	initial := Config{Version: 1, Backends: map[string]BackendProfile{}}
	if err := Save(root, initial); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	profile := BackendProfile{Kind: "codex", Executable: "/bin/echo"}
	if err := AddBackend(root, "codex-test", profile, false); err != nil {
		t.Fatalf("AddBackend() error = %v", err)
	}

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	got := loaded.Backends["codex-test"]
	if got.Kind != "codex" || got.Executable != "/bin/echo" {
		t.Fatalf("unexpected backend: %#v", got)
	}
}

func TestAddBackendRejectsDuplicateWithoutForce(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".sima"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Save(root, Config{Version: 1}); err != nil {
		t.Fatal(err)
	}
	profile := BackendProfile{Kind: "codex", Executable: "/bin/echo"}
	if err := AddBackend(root, "main", profile, false); err != nil {
		t.Fatal(err)
	}
	if err := AddBackend(root, "main", profile, false); err == nil {
		t.Fatal("expected duplicate add to fail")
	}
}
