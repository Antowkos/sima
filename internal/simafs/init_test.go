package simafs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesSimaScaffold(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	required := []string{
		".sima/config.yaml",
		".sima/schema.yaml",
		".sima/personal/memory/cards",
		".sima/personal/skills/active",
		".sima/team/memory/cards",
		".sima/system/skills/skill-authoring.md",
		".sima/system/prompts/archivist.md",
	}
	for _, rel := range required {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}
}

func TestDoctorFailsBeforeInitAndPassesAfterInit(t *testing.T) {
	root := t.TempDir()
	if Doctor(root).OK() {
		t.Fatal("Doctor() should fail before init")
	}
	if _, err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if !Doctor(root).OK() {
		t.Fatal("Doctor() should pass after init")
	}
}
