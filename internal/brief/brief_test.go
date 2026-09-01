package brief

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antowkos/sima/internal/config"
	"github.com/antowkos/sima/internal/simafs"
)

func TestGenerateWritesBriefWithSddArtifacts(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	planPath := filepath.Join(root, "docs", "plans", "test-plan.md")
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, []byte("# Test Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	memoryPath := filepath.Join(root, ".sima", "personal", "memory", "cards", "gotcha.yaml")
	if err := os.WriteFile(memoryPath, []byte("id: gotcha\nstatus: active\ntype: gotcha\ntitle: Remember active cards\ntrigger: When building a SIMA brief\nsummary: Active memory content should appear in the brief.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deprecatedMemoryPath := filepath.Join(root, ".sima", "personal", "memory", "cards", "deprecated.yaml")
	if err := os.WriteFile(deprecatedMemoryPath, []byte("id: deprecated\nstatus: deprecated\ntitle: Deprecated card\nsummary: Deprecated memory content must not appear in the brief.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(root, ".sima", "personal", "skills", "active", "brief-skill.md")
	if err := os.WriteFile(skillPath, []byte("---\nstatus: active\n---\n# Brief Skill\n\nUse when testing active skill snippets.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	supersededSkillPath := filepath.Join(root, ".sima", "personal", "skills", "active", "old-skill.md")
	if err := os.WriteFile(supersededSkillPath, []byte("---\nstatus: superseded\n---\n# Old Skill\n\nSuperseded skill content must not appear in the brief.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{Task: "testing active skill while building brief", Now: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("brief file missing: %v", err)
	}
	for _, want := range []string{"# SIMA Brief", "testing active skill while building brief", ".sima/system/skills/sdd-workflow.md", ".sima/personal/memory/cards/gotcha.yaml", "Remember active cards", "Active memory content should appear in the brief", ".sima/personal/skills/active/brief-skill.md", "Use when testing active skill snippets", "docs/plans/test-plan.md"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("brief missing %q:\n%s", want, result.Content)
		}
	}
	for _, unwanted := range []string{"Deprecated card", "Deprecated memory content must not appear", ".sima/personal/memory/cards/deprecated.yaml", "Old Skill", "Superseded skill content must not appear", ".sima/personal/skills/active/old-skill.md"} {
		if strings.Contains(result.Content, unwanted) {
			t.Fatalf("brief included inactive item %q:\n%s", unwanted, result.Content)
		}
	}
}

func TestGenerateFiltersActiveMemoryByTaskRelevance(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeRetrievalTestCards(t, root)

	result, err := Generate(root, Options{Task: "поправить guard-else-return в SupportConfig.swift", Now: time.Date(2026, 9, 1, 14, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	for _, want := range []string{"guard-before-return.yaml", "Guard before return styleguide rule"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("brief missing relevant memory %q:\n%s", want, result.Content)
		}
	}
	for _, unwanted := range []string{"gh-deployment.yaml", "GH deployment workflow", "android-core.yaml", "Android core repo reference"} {
		if strings.Contains(result.Content, unwanted) {
			t.Fatalf("brief included irrelevant memory %q:\n%s", unwanted, result.Content)
		}
	}
}

func TestGenerateCanUseExternalEmbeddingRetriever(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeRetrievalTestCards(t, root)
	script := filepath.Join(root, "fake-embed.py")
	if err := os.WriteFile(script, []byte(`#!/usr/bin/env python3
import json, sys
req = json.load(sys.stdin)
items = []
for item in req["texts"]:
    text = (item.get("text", "") + " " + item.get("path", "")).lower()
    if item["id"] == "__task__":
        vec = [1.0, 0.0]
    elif "android" in text:
        vec = [0.95, 0.0]
    elif "guard" in text:
        vec = [0.70, 0.0]
    else:
        vec = [0.0, 1.0]
    items.append({"id": item["id"], "vector": vec})
print(json.dumps({"embeddings": items}))
`), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Brief = config.Brief{Retrieval: "embedding", MaxSelected: 1, Embedding: config.BriefEmbedding{Command: "./fake-embed.py", Model: "intfloat/multilingual-e5-small", MinScore: 0.1}}
	cfg.LearnConfigured = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{Task: "поправить guard-else-return в SupportConfig.swift", Now: time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(result.Content, "android-core.yaml") {
		t.Fatalf("embedding scorer top result was not used:\n%s", result.Content)
	}
	if strings.Contains(result.Content, "guard-before-return.yaml") || strings.Contains(result.Content, "gh-deployment.yaml") {
		t.Fatalf("embedding max_selected=1 should include only top scorer:\n%s", result.Content)
	}
}

func TestGenerateFallsBackToLexicalWhenEmbeddingFails(t *testing.T) {
	root := t.TempDir()
	if _, err := simafs.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeRetrievalTestCards(t, root)
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Brief = config.Brief{Retrieval: "hybrid", Embedding: config.BriefEmbedding{Command: "./missing-embedder"}}
	cfg.LearnConfigured = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}

	result, err := Generate(root, Options{Task: "поправить guard-else-return в SupportConfig.swift", Now: time.Date(2026, 9, 1, 16, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(result.Content, "guard-before-return.yaml") || strings.Contains(result.Content, "android-core.yaml") {
		t.Fatalf("hybrid fallback should use lexical relevance:\n%s", result.Content)
	}
}

func writeRetrievalTestCards(t *testing.T, root string) {
	t.Helper()
	cards := map[string]string{
		"guard-before-return.yaml": "id: guard-before-return\nstatus: active\ntype: invariant\ntitle: Guard before return styleguide rule\ntrigger: When editing Swift functions with guard-else-return style\nsummary: Prefer an early guard check before returning optional values from Swift functions.\n",
		"gh-deployment.yaml":       "id: gh-deployment\nstatus: active\ntype: workflow\ntitle: GH deployment workflow\ntrigger: When running the deployment workflow from GitHub CLI\nsummary: Use gh workflow run Run deployment with the target environment input.\n",
		"android-core.yaml":        "id: android-core\nstatus: active\ntype: invariant\ntitle: Android core repo reference\ntrigger: When working with the Android sibling repository\nsummary: The sibling android-core repository lives in the adjacent checkout path.\n",
	}
	for name, content := range cards {
		if err := os.WriteFile(filepath.Join(root, ".sima", "personal", "memory", "cards", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGenerateRequiresTask(t *testing.T) {
	_, err := Generate(t.TempDir(), Options{})
	if err == nil {
		t.Fatal("expected missing task error")
	}
}
