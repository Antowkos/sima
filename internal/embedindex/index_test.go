package embedindex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antowkos/sima/internal/config"
	"github.com/antowkos/sima/internal/simafs"
)

func TestRebuildIndexesActiveKnowledgeAndSkipsInactive(t *testing.T) {
	root := setupProjectWithEmbedder(t)
	writeCard(t, root, "guard.yaml", "active", "Guard rule", "When editing Swift guard returns", "Use guard before return.")
	writeCard(t, root, "old.yaml", "deprecated", "Old rule", "When doing old work", "Do not index this.")

	cfg := embeddingConfig()
	result, err := Rebuild(root, cfg)
	if err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}
	if result.Indexed != 1 {
		t.Fatalf("Indexed = %d, want 1", result.Indexed)
	}
	entries, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !strings.Contains(entries[0].Path, "guard.yaml") || len(entries[0].Vector) == 0 {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestSelectRelevantRefreshesStaleEditedCards(t *testing.T) {
	root := setupProjectWithEmbedder(t)
	path := writeCard(t, root, "guard.yaml", "active", "Guard rule", "When editing Swift guard returns", "Use guard before return.")
	cfg := embeddingConfig()
	if _, err := Rebuild(root, cfg); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}
	before, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("id: guard\nstatus: active\ntype: invariant\ntitle: Android repo rule\ntrigger: When editing Android modules\nsummary: Use the android-core repository.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	selected, ok := SelectRelevant(root, []string{".sima/personal/memory/cards/guard.yaml"}, "android module", cfg)
	if !ok || len(selected) != 1 || selected[0] != ".sima/personal/memory/cards/guard.yaml" {
		t.Fatalf("SelectRelevant() = %#v, %v", selected, ok)
	}
	after, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if before[0].TextHash == after[0].TextHash {
		t.Fatalf("stale edited card hash was not refreshed")
	}
}

func setupProjectWithEmbedder(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "fake-embed.py")
	if err := os.WriteFile(script, []byte(`#!/usr/bin/env python3
import hashlib, json, sys
req = json.load(sys.stdin)
out = []
for item in req["texts"]:
    text = (item.get("text", "") + " " + item.get("path", "")).lower()
    if "android" in text:
        vec = [1.0, 0.0]
    elif "guard" in text:
        vec = [0.8, 0.0]
    else:
        # deterministic non-zero fallback
        h = int(hashlib.sha256(text.encode()).hexdigest()[:8], 16) / 0xffffffff
        vec = [0.0, 0.5 + h]
    out.append({"id": item["id"], "vector": vec})
print(json.dumps({"embeddings": out}))
`), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func embeddingConfig() config.Brief {
	return config.Brief{Retrieval: "embedding", MaxSelected: 8, Embedding: config.BriefEmbedding{Command: "./fake-embed.py", Model: "test-embed", MinScore: 0.1}}
}

func writeCard(t *testing.T, root, name, status, title, trigger, summary string) string {
	t.Helper()
	path := filepath.Join(root, ".sima", "personal", "memory", "cards", name)
	content := "id: " + strings.TrimSuffix(name, filepath.Ext(name)) + "\nstatus: " + status + "\ntype: invariant\ntitle: " + title + "\ntrigger: " + trigger + "\nsummary: " + summary + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
