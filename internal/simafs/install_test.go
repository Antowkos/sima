package simafs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestInstallInstructionsWritesManagedBlocks(t *testing.T) {
	root := t.TempDir()
	result, err := InstallInstructions(root, InstallOptions{})
	if err != nil {
		t.Fatalf("InstallInstructions() error = %v", err)
	}
	for _, want := range []string{"CLAUDE.md", "AGENTS.md", ".claude/commands/sima.md", ".claude/commands/sima-brief.md", ".claude/commands/sima-remember.md", ".codex/skills/sima/SKILL.md", ".codex/skills/sima-brief/SKILL.md", ".codex/skills/sima-remember/SKILL.md"} {
		if !containsString(result.Written, want) {
			t.Fatalf("written files missing %s: %#v", want, result.Written)
		}
	}
	for _, rel := range []string{"CLAUDE.md", "AGENTS.md"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, want := range []string{managedBlockStart, "sima brief \"<task>\" --path .", "sima learn --backend <backend-name> --task \"<task>\" --path .", "Do not learn: transient task progress", managedBlockEnd} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %q:\n%s", rel, want, text)
			}
		}
	}
	for _, rel := range []string{".claude/commands/sima.md", ".claude/commands/sima-brief.md", ".claude/commands/sima-remember.md"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, want := range []string{"description:", "$ARGUMENTS", "sima"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %q:\n%s", rel, want, text)
			}
		}
	}
	for _, rel := range []string{".codex/skills/sima/SKILL.md", ".codex/skills/sima-brief/SKILL.md", ".codex/skills/sima-remember/SKILL.md"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, want := range []string{"description:", "name: sima", "sima"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %q:\n%s", rel, want, text)
			}
		}
	}
}

func TestInstallInstructionsPreservesAndReplacesManagedBlock(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "CLAUDE.md")
	original := "# Existing Instructions\n\nKeep this.\n\n" + managedBlockStart + "\nold\n" + managedBlockEnd + "\n\nFooter.\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallInstructions(root, InstallOptions{Clients: []string{"claude"}}); err != nil {
		t.Fatalf("InstallInstructions() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "Keep this.") || !strings.Contains(text, "Footer.") {
		t.Fatalf("existing content not preserved:\n%s", text)
	}
	if strings.Contains(text, "\nold\n") {
		t.Fatalf("old managed block was not replaced:\n%s", text)
	}
	if strings.Count(text, managedBlockStart) != 1 || strings.Count(text, managedBlockEnd) != 1 {
		t.Fatalf("managed block duplicated:\n%s", text)
	}
}
