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

func TestSelectRelevantDefaultMinScoreFiltersWeakMatches(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatal(err)
	}
	writeCard(t, root, "strong.yaml", "active", "SwiftUI profile screen", "When building profile UI", "Use the profile view model.")
	writeCard(t, root, "weak.yaml", "active", "Deployment workflow", "When deploying backend", "Use gh deployment checks.")
	script := filepath.Join(root, "threshold-embed.py")
	if err := os.WriteFile(script, []byte(`#!/usr/bin/env python3
import json, sys
req = json.load(sys.stdin)
out = []
for item in req["texts"]:
    ident = item["id"]
    text = (item.get("text", "") + " " + item.get("path", "")).lower()
    if ident == "__task__":
        vec = [1.0, 0.0]
    elif "strong" in text:
        vec = [0.9, 0.4358898944]
    else:
        vec = [0.84, 0.5425863987]
    out.append({"id": ident, "vector": vec})
print(json.dumps({"embeddings": out}))
`), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Brief{Retrieval: "embedding", MaxSelected: 8, Embedding: config.BriefEmbedding{Command: "./threshold-embed.py", Model: "test-embed"}}
	if _, err := Rebuild(root, cfg); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}
	selected, ok := SelectRelevant(root, []string{".sima/personal/memory/cards/strong.yaml", ".sima/personal/memory/cards/weak.yaml"}, "build SwiftUI profile screen", cfg)
	if !ok || len(selected) != 1 || selected[0] != ".sima/personal/memory/cards/strong.yaml" {
		t.Fatalf("default min score should keep strong match and drop weak baseline match: %#v ok=%v", selected, ok)
	}
}

func TestSelectRelevantCommandDecompositionRecoversMultiTopicMatches(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatal(err)
	}
	writeCard(t, root, "guard.yaml", "active", "Guard before return", "When editing guard else return", "Keep guard before return style.")
	writeCard(t, root, "pr.yaml", "active", "PR title and body format", "When opening a pull request", "Use the repository pull request format.")
	writeCard(t, root, "deploy.yaml", "active", "Deployment workflow", "When deploying backend", "Use deployment checks.")

	embedScript := filepath.Join(root, "topic-embed.py")
	if err := os.WriteFile(embedScript, []byte(`#!/usr/bin/env python3
import json, sys
req = json.load(sys.stdin)
out = []
for item in req["texts"]:
    ident = item["id"]
    text = (item.get("text", "") + " " + item.get("path", "")).lower()
    if ident == "__task__":
        vec = [0.6, 0.6]
    elif "guard" in text:
        vec = [1.0, 0.0]
    elif "pull request" in text or "pr" in text:
        vec = [0.0, 1.0]
    elif "deploy" in text:
        vec = [-1.0, 0.0]
    else:
        vec = [0.0, -1.0]
    out.append({"id": ident, "vector": vec})
print(json.dumps({"embeddings": out}))
`), 0o755); err != nil {
		t.Fatal(err)
	}
	splitScript := filepath.Join(root, "split-query.py")
	if err := os.WriteFile(splitScript, []byte(`#!/usr/bin/env python3
import json, sys
req = json.load(sys.stdin)
task = req.get("task", "").lower()
queries = []
if "guard" in task and "pr" in task:
    queries = ["исправить guard-else-return", "открыть PR"]
print(json.dumps({"queries": queries[: req.get("max_parts", 4)]}))
`), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.Brief{
		Retrieval:   "embedding",
		MaxSelected: 8,
		Embedding:   config.BriefEmbedding{Command: "./topic-embed.py", Model: "test-embed", MinScore: 0.85},
	}
	paths := []string{".sima/personal/memory/cards/guard.yaml", ".sima/personal/memory/cards/pr.yaml", ".sima/personal/memory/cards/deploy.yaml"}
	if _, err := Rebuild(root, cfg); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}
	selected, ok := SelectRelevant(root, paths, "исправить guard-else-return и открыть PR", cfg)
	if !ok || len(selected) != 0 {
		t.Fatalf("without decomposition diluted multi-topic query should complete with no matches: %#v ok=%v", selected, ok)
	}

	cfg.Query = config.BriefQuery{Decomposition: "command", Command: "./split-query.py", MaxParts: 4}
	selected, ok = SelectRelevant(root, paths, "исправить guard-else-return и открыть PR", cfg)
	if !ok || !sameStringSet(selected, []string{".sima/personal/memory/cards/guard.yaml", ".sima/personal/memory/cards/pr.yaml"}) {
		t.Fatalf("command decomposition should recover both relevant topic cards without deploy noise: %#v ok=%v", selected, ok)
	}

	selected, ok = SelectRelevant(root, paths, "написать unit-тест для парсера ответов API", cfg)
	if !ok || len(selected) != 0 {
		t.Fatalf("unrelated query should complete with no matches when decomposition is enabled: %#v ok=%v", selected, ok)
	}
}

func TestSelectRelevantAnchoredPartTopKRescuesShortSubqueryWithoutUnrelatedNoise(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatal(err)
	}
	writeCard(t, root, "guard.yaml", "active", "Guard before return", "When editing guard else return", "Keep guard before return style.")
	writeCard(t, root, "pr.yaml", "active", "PR title and body format", "When opening a pull request", "Use the repository pull request format.")
	writeCard(t, root, "deploy.yaml", "active", "Deployment workflow", "When deploying backend", "Use deployment checks.")

	script := filepath.Join(root, "reported-scores-embed.py")
	if err := os.WriteFile(script, []byte(`#!/usr/bin/env python3
import math, json, sys
req = json.load(sys.stdin)
out = []
for item in req["texts"]:
    ident = item["id"]
    text = (item.get("text", "") + " " + item.get("path", "")).lower()
    if ident == "__task__" and "unit" in text:
        vec = [math.sqrt(1 - 0.8385**2), 0.8385]
    elif ident == "__task__" and "guard" in text and "pr" in text:
        vec = [0.9, math.sqrt(1 - 0.9**2)]
    elif ident.startswith("__task_part_") and "guard" in text:
        vec = [1.0, 0.0]
    elif ident.startswith("__task_part_") and "pr" in text:
        vec = [math.sqrt(1 - 0.808**2), 0.808]
    elif "guard" in text:
        vec = [1.0, 0.0]
    elif "pull request" in text or "pr" in text:
        vec = [0.0, 1.0]
    elif "deploy" in text:
        vec = [-1.0, 0.0]
    else:
        vec = [0.0, -1.0]
    out.append({"id": ident, "vector": vec})
print(json.dumps({"embeddings": out}))
`), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.Brief{
		Retrieval:   "embedding",
		MaxSelected: 8,
		Embedding:   config.BriefEmbedding{Command: "./reported-scores-embed.py", Model: "test-embed", MinScore: 0.85},
		Query:       config.BriefQuery{Decomposition: "heuristic", MaxParts: 4, TopKPerPart: 1, MinPartScore: 0.80},
	}
	paths := []string{".sima/personal/memory/cards/guard.yaml", ".sima/personal/memory/cards/pr.yaml", ".sima/personal/memory/cards/deploy.yaml"}
	if _, err := Rebuild(root, cfg); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	selected, ok := SelectRelevant(root, paths, "исправить guard-else-return и открыть PR", cfg)
	if !ok || len(selected) != 2 || selected[0] != ".sima/personal/memory/cards/guard.yaml" || selected[1] != ".sima/personal/memory/cards/pr.yaml" {
		t.Fatalf("anchored part top-k should rescue short PR subquery with 0.808 score: %#v ok=%v", selected, ok)
	}

	selected, ok = SelectRelevant(root, paths, "написать unit-тест для парсера ответов API", cfg)
	if !ok || len(selected) != 0 {
		t.Fatalf("unrelated 0.8385 PR noise should not be rescued without a high-confidence anchor: %#v ok=%v", selected, ok)
	}
}

func TestSelectRelevantWholeTaskTopKRescueRequiresLexicalSupport(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatal(err)
	}
	writeCard(t, root, "pr.yaml", "active", "PR title and body format", "When opening or drafting a pull request in this repo", "Use the repository pull request format.")
	writeCard(t, root, "android.yaml", "active", "Android core repo reference", "When the user references Android behavior for comparison while working on an iOS feature", "Compare feature-flag patterns and domain models across platforms.")
	writeCard(t, root, "deploy.yaml", "active", "Deployment workflow", "When deploying backend", "Use deployment checks.")

	script := filepath.Join(root, "single-topic-scores.py")
	if err := os.WriteFile(script, []byte(`#!/usr/bin/env python3
import math, json, sys
req = json.load(sys.stdin)
out = []
for item in req["texts"]:
    ident = item["id"]
    text = (item.get("text", "") + " " + item.get("path", "")).lower()
    if ident == "__task__" and "unit" in text:
        vec = [math.sqrt(1 - 0.8385**2), 0.8385, 0.0]
    elif ident == "__task__" and "feature/x" in text:
        vec = [math.sqrt(1 - 0.8385**2), 0.0, 0.8385]
    elif ident == "__task__" and "pr" in text:
        vec = [math.sqrt(1 - 0.8299**2), 0.8299, 0.0]
    elif ident == "__task__" and "android" in text:
        vec = [math.sqrt(1 - 0.8497**2), 0.0, 0.8497]
    elif "pull request" in text or "pr" in text:
        vec = [0.0, 1.0, 0.0]
    elif "android" in text or "feature-flag" in text:
        vec = [0.0, 0.0, 1.0]
    elif "deploy" in text:
        vec = [-1.0, 0.0, 0.0]
    else:
        vec = [0.0, -1.0, 0.0]
    out.append({"id": ident, "vector": vec})
print(json.dumps({"embeddings": out}))
`), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.Brief{
		Retrieval:   "embedding",
		MaxSelected: 8,
		Embedding:   config.BriefEmbedding{Command: "./single-topic-scores.py", Model: "test-embed", MinScore: 0.85},
		Query:       config.BriefQuery{Decomposition: "heuristic", MaxParts: 4, TopKPerPart: 1, MinPartScore: 0.80},
	}
	paths := []string{".sima/personal/memory/cards/pr.yaml", ".sima/personal/memory/cards/android.yaml", ".sima/personal/memory/cards/deploy.yaml"}
	if _, err := Rebuild(root, cfg); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	selected, ok := SelectRelevant(root, paths, "открыть PR для новой фичи BUSDEL-700", cfg)
	if !ok || len(selected) != 1 || selected[0] != ".sima/personal/memory/cards/pr.yaml" {
		t.Fatalf("whole-task rescue should recover PR match below global threshold: %#v ok=%v", selected, ok)
	}

	selected, ok = SelectRelevant(root, paths, "сверить фичу с реализацией на android", cfg)
	if !ok || len(selected) != 1 || selected[0] != ".sima/personal/memory/cards/android.yaml" {
		t.Fatalf("whole-task rescue should recover Android match below global threshold: %#v ok=%v", selected, ok)
	}

	selected, ok = SelectRelevant(root, paths, "написать unit-тест для парсера ответов API", cfg)
	if !ok || len(selected) != 0 {
		t.Fatalf("whole-task rescue should not recover unrelated PR noise without lexical support: %#v ok=%v", selected, ok)
	}

	selected, ok = SelectRelevant(root, paths, "задеплоить bus приложение с ветки feature/x", cfg)
	if !ok || len(selected) != 0 {
		t.Fatalf("whole-task rescue should not use generic feature token as lexical support: %#v ok=%v", selected, ok)
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

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	counts := map[string]int{}
	for _, item := range got {
		counts[item]++
	}
	for _, item := range want {
		counts[item]--
		if counts[item] < 0 {
			return false
		}
	}
	return true
}
