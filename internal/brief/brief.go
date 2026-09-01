package brief

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/antowkos/sima/internal/config"
)

const (
	maxSnippetBytes = 900
	maxSnippetItems = 12
)

type Result struct {
	Path    string
	Content string
}

type Options struct {
	Task string
	Now  time.Time
}

type snippet struct {
	Path    string
	Content string
}

func Generate(projectRoot string, opts Options) (Result, error) {
	if strings.TrimSpace(opts.Task) == "" {
		return Result{}, fmt.Errorf("task is required")
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}

	simaRoot := filepath.Join(projectRoot, ".sima")
	briefCfg := config.Brief{Retrieval: "lexical"}
	if cfg, err := config.Load(projectRoot); err == nil {
		briefCfg = cfg.Brief
		if briefCfg.Retrieval == "" {
			briefCfg.Retrieval = "lexical"
		}
	}
	briefDir := filepath.Join(simaRoot, "personal", "briefs")
	if err := os.MkdirAll(briefDir, 0o755); err != nil {
		return Result{}, err
	}

	personalMemory := readRelevantActiveSnippets(filepath.Join(simaRoot, "personal", "memory", "cards"), projectRoot, []string{".yaml", ".yml", ".md"}, opts.Task, briefCfg)
	personalSkills := readRelevantActiveSnippets(filepath.Join(simaRoot, "personal", "skills", "active"), projectRoot, []string{".md"}, opts.Task, briefCfg)
	teamMemory := readRelevantActiveSnippets(filepath.Join(simaRoot, "team", "memory", "cards"), projectRoot, []string{".yaml", ".yml", ".md"}, opts.Task, briefCfg)
	teamSkills := readRelevantActiveSnippets(filepath.Join(simaRoot, "team", "skills", "active"), projectRoot, []string{".md"}, opts.Task, briefCfg)

	data := data{
		Task:                   opts.Task,
		GeneratedAt:            opts.Now.UTC().Format(time.RFC3339),
		SystemSkills:           listFiles(filepath.Join(simaRoot, "system", "skills"), projectRoot, []string{".md"}),
		PersonalMemorySnippets: personalMemory,
		PersonalSkillSnippets:  personalSkills,
		TeamMemorySnippets:     teamMemory,
		TeamSkillSnippets:      teamSkills,
		SDDArtifacts:           findSDDArtifacts(projectRoot),
	}
	content := render(data)
	name := opts.Now.UTC().Format("20060102-150405") + "-brief.md"
	path := filepath.Join(briefDir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return Result{}, err
	}
	return Result{Path: path, Content: content}, nil
}

type data struct {
	Task                   string
	GeneratedAt            string
	SystemSkills           []string
	PersonalMemory         []string
	PersonalMemorySnippets []snippet
	PersonalSkills         []string
	PersonalSkillSnippets  []snippet
	TeamMemory             []string
	TeamMemorySnippets     []snippet
	TeamSkills             []string
	TeamSkillSnippets      []snippet
	SDDArtifacts           []string
}

func render(d data) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# SIMA Brief\n\n")
	fmt.Fprintf(&b, "**Generated:** %s\n\n", d.GeneratedAt)
	fmt.Fprintf(&b, "## Task\n\n%s\n\n", d.Task)
	writeSection(&b, "System skills", d.SystemSkills, "No system skills found. Run `sima init`.")
	writeSnippetSection(&b, "Personal memory", d.PersonalMemorySnippets, "No active personal memory cards.")
	writeSnippetSection(&b, "Personal skills", d.PersonalSkillSnippets, "No active personal skills.")
	writeSnippetSection(&b, "Team/shared memory", d.TeamMemorySnippets, "No active team memory cards. Team scope is scaffolded for later.")
	writeSnippetSection(&b, "Team/shared skills", d.TeamSkillSnippets, "No active team skills. Team scope is scaffolded for later.")
	writeSection(&b, "SDD source artifacts", d.SDDArtifacts, "No SDD artifacts found under docs/specs, docs/plans, or openspec/changes.")
	fmt.Fprintf(&b, "## Policy\n\n")
	fmt.Fprintf(&b, "- Treat SDD specs/plans as source artifacts, not active memory.\n")
	fmt.Fprintf(&b, "- Learn only durable lessons with evidence pointers.\n")
	fmt.Fprintf(&b, "- Do not weaken tests, bypass validation, hardcode outputs, or remove obstacles instead of fixing root causes.\n")
	fmt.Fprintf(&b, "- Archivist must run in a clean separate session before auto-applying personal memory/skill changes.\n")
	return b.String()
}

func writeSection(b *strings.Builder, title string, items []string, empty string) {
	fmt.Fprintf(b, "## %s\n\n", title)
	if len(items) == 0 {
		fmt.Fprintf(b, "%s\n\n", empty)
		return
	}
	for _, item := range items {
		fmt.Fprintf(b, "- `%s`\n", item)
	}
	fmt.Fprintln(b)
}

func writeSnippetSection(b *strings.Builder, title string, items []snippet, empty string) {
	fmt.Fprintf(b, "## %s\n\n", title)
	if len(items) == 0 {
		fmt.Fprintf(b, "%s\n\n", empty)
		return
	}
	for _, item := range items {
		fmt.Fprintf(b, "### `%s`\n\n", item.Path)
		fmt.Fprintf(b, "```text\n%s\n```\n\n", item.Content)
	}
}

func listFiles(root, projectRoot string, exts []string) []string {
	var items []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if hasExt(path, exts) {
			items = append(items, rel(projectRoot, path))
		}
		return nil
	})
	sort.Strings(items)
	return items
}

func listActiveFiles(root, projectRoot string, exts []string) []string {
	paths := listFiles(root, projectRoot, exts)
	active := paths[:0]
	for _, relPath := range paths {
		abs := filepath.Join(projectRoot, filepath.FromSlash(relPath))
		if activeKnowledgeFile(abs) {
			active = append(active, relPath)
		}
	}
	return active
}

func readSnippets(root, projectRoot string, exts []string) []snippet {
	paths := listFiles(root, projectRoot, exts)
	if len(paths) > maxSnippetItems {
		paths = paths[:maxSnippetItems]
	}
	items := make([]snippet, 0, len(paths))
	for _, relPath := range paths {
		abs := filepath.Join(projectRoot, filepath.FromSlash(relPath))
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if len(content) > maxSnippetBytes {
			content = strings.TrimSpace(content[:maxSnippetBytes]) + "\n... [truncated]"
		}
		items = append(items, snippet{Path: relPath, Content: content})
	}
	return items
}

func readActiveSnippets(root, projectRoot string, exts []string) []snippet {
	paths := listActiveFiles(root, projectRoot, exts)
	return snippetsForPaths(projectRoot, paths)
}

func listRelevantActiveFiles(root, projectRoot string, exts []string, task string, cfg config.Brief) []string {
	paths := listActiveFiles(root, projectRoot, exts)
	if selected, ok := embeddingRelevantActiveFiles(projectRoot, paths, task, cfg); ok {
		return selected
	}
	relevant := paths[:0]
	for _, relPath := range paths {
		abs := filepath.Join(projectRoot, filepath.FromSlash(relPath))
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		if relevantToTask(task, relPath, string(data)) {
			relevant = append(relevant, relPath)
		}
	}
	return relevant
}

func readRelevantActiveSnippets(root, projectRoot string, exts []string, task string, cfg config.Brief) []snippet {
	paths := listRelevantActiveFiles(root, projectRoot, exts, task, cfg)
	return snippetsForPaths(projectRoot, paths)
}

func snippetsForPaths(projectRoot string, paths []string) []snippet {
	if len(paths) > maxSnippetItems {
		paths = paths[:maxSnippetItems]
	}
	items := make([]snippet, 0, len(paths))
	for _, relPath := range paths {
		abs := filepath.Join(projectRoot, filepath.FromSlash(relPath))
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if len(content) > maxSnippetBytes {
			content = strings.TrimSpace(content[:maxSnippetBytes]) + "\n... [truncated]"
		}
		items = append(items, snippet{Path: relPath, Content: content})
	}
	return items
}

func relevantToTask(task, relPath, content string) bool {
	// Alpha lexical fallback is intentionally deterministic and dependency-free:
	// match task tokens against path/title/trigger/summary tokens. This prevents
	// whole active-memory dumps. Projects can opt into embedding retrieval through
	// brief.embedding.command; SIMA invokes that command per brief and does not
	// keep the model resident unless the configured command talks to a daemon.
	taskTokens := meaningfulTokens(task)
	if len(taskTokens) == 0 {
		return false
	}
	candidate := relPath + "\n" + relevanceText(content)
	for token := range meaningfulTokens(candidate) {
		if taskTokens[token] {
			return true
		}
	}
	return false
}

type embeddingRequest struct {
	Model string          `json:"model,omitempty"`
	Texts []embeddingText `json:"texts"`
}

type embeddingText struct {
	ID   string `json:"id"`
	Path string `json:"path,omitempty"`
	Text string `json:"text"`
}

type embeddingResponse struct {
	Embeddings []embeddingVector `json:"embeddings"`
}

type embeddingVector struct {
	ID     string    `json:"id"`
	Vector []float64 `json:"vector"`
}

func embeddingRelevantActiveFiles(projectRoot string, paths []string, task string, cfg config.Brief) ([]string, bool) {
	retrieval := strings.ToLower(strings.TrimSpace(cfg.Retrieval))
	if retrieval != "embedding" && retrieval != "hybrid" {
		return nil, false
	}
	command := strings.TrimSpace(cfg.Embedding.Command)
	if command == "" || len(paths) == 0 {
		return nil, false
	}
	texts := []embeddingText{{ID: "__task__", Text: task}}
	for _, relPath := range paths {
		abs := filepath.Join(projectRoot, filepath.FromSlash(relPath))
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		texts = append(texts, embeddingText{ID: relPath, Path: relPath, Text: relevanceText(string(data))})
	}
	vectors, err := runEmbeddingCommand(projectRoot, command, cfg.Embedding.Model, texts)
	if err != nil {
		return nil, false
	}
	taskVec, ok := vectors["__task__"]
	if !ok || len(taskVec) == 0 {
		return nil, false
	}
	maxSelected := cfg.MaxSelected
	if maxSelected <= 0 || maxSelected > maxSnippetItems {
		maxSelected = maxSnippetItems
	}
	minScore := cfg.Embedding.MinScore
	if minScore == 0 {
		minScore = 0.20
	}
	type scored struct {
		path  string
		score float64
	}
	var scoredItems []scored
	for _, relPath := range paths {
		vec, ok := vectors[relPath]
		if !ok {
			continue
		}
		score := cosine(taskVec, vec)
		if score >= minScore {
			scoredItems = append(scoredItems, scored{path: relPath, score: score})
		}
	}
	sort.SliceStable(scoredItems, func(i, j int) bool { return scoredItems[i].score > scoredItems[j].score })
	if len(scoredItems) > maxSelected {
		scoredItems = scoredItems[:maxSelected]
	}
	selected := make([]string, 0, len(scoredItems))
	for _, item := range scoredItems {
		selected = append(selected, item.path)
	}
	return selected, len(selected) > 0
}

func runEmbeddingCommand(projectRoot, command, model string, texts []embeddingText) (map[string][]float64, error) {
	payload, err := json.Marshal(embeddingRequest{Model: model, Texts: texts})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = projectRoot
	cmd.Stdin = bytes.NewReader(payload)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var response embeddingResponse
	if err := json.Unmarshal(bytes.TrimSpace(out), &response); err != nil {
		return nil, err
	}
	vectors := map[string][]float64{}
	for _, item := range response.Embeddings {
		if item.ID != "" && len(item.Vector) > 0 {
			vectors[item.ID] = item.Vector
		}
	}
	return vectors, nil
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return -1
	}
	var dot, aa, bb float64
	for i := range a {
		dot += a[i] * b[i]
		aa += a[i] * a[i]
		bb += b[i] * b[i]
	}
	if aa == 0 || bb == 0 {
		return -1
	}
	return dot / (math.Sqrt(aa) * math.Sqrt(bb))
}

func relevanceText(content string) string {
	var meta struct {
		ID      string `yaml:"id"`
		Type    string `yaml:"type"`
		Title   string `yaml:"title"`
		Trigger string `yaml:"trigger"`
		Summary string `yaml:"summary"`
		Status  string `yaml:"status"`
	}
	if err := yaml.Unmarshal([]byte(content), &meta); err == nil && (meta.Title != "" || meta.Trigger != "" || meta.Summary != "") {
		return strings.Join([]string{meta.ID, meta.Type, meta.Title, meta.Trigger, meta.Summary, meta.Status}, "\n")
	}
	return content
}

func meaningfulTokens(text string) map[string]bool {
	tokens := make(map[string]bool)
	for _, field := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= 'а' && r <= 'я' || r == 'ё')
	}) {
		field = strings.TrimSpace(field)
		if len([]rune(field)) < 3 || stopToken(field) {
			continue
		}
		tokens[field] = true
		if strings.HasSuffix(field, "ing") && len(field) > 5 {
			tokens[strings.TrimSuffix(field, "ing")] = true
		}
	}
	return tokens
}

func stopToken(token string) bool {
	switch token {
	case "when", "while", "before", "after", "during", "this", "that", "with", "from", "into", "project", "repo", "task", "work", "working", "active", "memory", "skill", "sima", "brief", "user", "source", "status", "title", "trigger", "summary", "type", "kind", "the", "and", "for", "или", "для", "при", "про", "это", "как", "что", "когда", "если", "надо", "нужно", "проект", "репо", "задача", "память", "скил", "активный":
		return true
	default:
		return false
	}
}

func activeKnowledgeFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	status := statusFromContent(path, string(data))
	return status == "active"
}

func statusFromContent(path, content string) string {
	if hasExt(path, []string{".yaml", ".yml"}) {
		var meta struct {
			Status string `yaml:"status"`
		}
		if err := yaml.Unmarshal([]byte(content), &meta); err != nil {
			return ""
		}
		return strings.TrimSpace(strings.ToLower(meta.Status))
	}
	return strings.TrimSpace(strings.ToLower(frontmatterValue(content, "status")))
}

func frontmatterValue(content, key string) string {
	if !strings.HasPrefix(content, "---\n") {
		return ""
	}
	parts := strings.SplitN(content, "\n---\n", 2)
	if len(parts) != 2 {
		return ""
	}
	prefix := key + ":"
	for _, line := range strings.Split(parts[0], "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func findSDDArtifacts(projectRoot string) []string {
	roots := []string{
		filepath.Join(projectRoot, "docs", "specs"),
		filepath.Join(projectRoot, "docs", "plans"),
		filepath.Join(projectRoot, "openspec", "changes"),
	}
	var items []string
	for _, root := range roots {
		items = append(items, listFiles(root, projectRoot, []string{".md"})...)
	}
	sort.Strings(items)
	return items
}

func hasExt(path string, exts []string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, candidate := range exts {
		if ext == candidate {
			return true
		}
	}
	return false
}

func rel(projectRoot, path string) string {
	rel, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
