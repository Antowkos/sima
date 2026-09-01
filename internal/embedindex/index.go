package embedindex

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

const indexRelPath = ".sima/index/embeddings.jsonl"
const defaultMinScore = 0.85

// Entry is one persisted embedding for active SIMA knowledge metadata.
type Entry struct {
	Path      string    `json:"path"`
	Kind      string    `json:"kind"`
	Scope     string    `json:"scope"`
	Model     string    `json:"model,omitempty"`
	TextHash  string    `json:"text_hash"`
	Vector    []float64 `json:"vector"`
	UpdatedAt string    `json:"updated_at"`
}

type Result struct {
	Path    string
	Indexed int
	Removed int
	Skipped int
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

type queryDecompositionRequest struct {
	Task     string `json:"task"`
	MaxParts int    `json:"max_parts,omitempty"`
}

type queryDecompositionResponse struct {
	Queries []string `json:"queries"`
}

type embeddingVector struct {
	ID     string    `json:"id"`
	Vector []float64 `json:"vector"`
}

type Candidate struct {
	Path     string
	Kind     string
	Scope    string
	Text     string
	TextHash string
}

type scoredItem struct {
	path  string
	score float64
}

func IndexPath(projectRoot string) string {
	return filepath.Join(projectRoot, filepath.FromSlash(indexRelPath))
}

func Enabled(cfg config.Brief) bool {
	retrieval := strings.ToLower(strings.TrimSpace(cfg.Retrieval))
	return (retrieval == "embedding" || retrieval == "hybrid") && strings.TrimSpace(cfg.Embedding.Command) != ""
}

func Rebuild(projectRoot string, cfg config.Brief) (Result, error) {
	if !Enabled(cfg) {
		return Result{Path: rel(projectRoot, IndexPath(projectRoot))}, fmt.Errorf("brief.embedding.command is required and brief.retrieval must be embedding or hybrid")
	}
	candidates := DiscoverActive(projectRoot)
	entries, skipped, err := embedCandidates(projectRoot, cfg, candidates)
	if err != nil {
		return Result{}, err
	}
	if err := writeEntries(projectRoot, entries); err != nil {
		return Result{}, err
	}
	return Result{Path: rel(projectRoot, IndexPath(projectRoot)), Indexed: len(entries), Removed: 0, Skipped: skipped}, nil
}

func UpdatePaths(projectRoot string, cfg config.Brief, paths []string) (Result, error) {
	if !Enabled(cfg) {
		return Result{Path: rel(projectRoot, IndexPath(projectRoot))}, nil
	}
	existing, err := Read(projectRoot)
	if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	pathSet := map[string]bool{}
	for _, p := range paths {
		if p = cleanRel(p); p != "" {
			pathSet[p] = true
		}
	}
	kept := make([]Entry, 0, len(existing))
	removed := 0
	for _, entry := range existing {
		if pathSet[entry.Path] {
			removed++
			continue
		}
		kept = append(kept, entry)
	}
	var candidates []Candidate
	for p := range pathSet {
		if c, ok := candidateForPath(projectRoot, p); ok {
			candidates = append(candidates, c)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Path < candidates[j].Path })
	newEntries, skipped, err := embedCandidates(projectRoot, cfg, candidates)
	if err != nil {
		return Result{}, err
	}
	all := append(kept, newEntries...)
	sort.Slice(all, func(i, j int) bool { return all[i].Path < all[j].Path })
	if err := writeEntries(projectRoot, all); err != nil {
		return Result{}, err
	}
	return Result{Path: rel(projectRoot, IndexPath(projectRoot)), Indexed: len(newEntries), Removed: removed, Skipped: skipped}, nil
}

func SelectRelevant(projectRoot string, paths []string, task string, cfg config.Brief) ([]string, bool) {
	if !Enabled(cfg) || len(paths) == 0 {
		return nil, false
	}
	current := make([]Candidate, 0, len(paths))
	for _, p := range paths {
		if c, ok := candidateForPath(projectRoot, p); ok {
			current = append(current, c)
		}
	}
	if len(current) == 0 {
		return nil, false
	}
	entries, err := Read(projectRoot)
	if err != nil && !os.IsNotExist(err) {
		return nil, false
	}
	stale := stalePaths(current, entries, cfg.Embedding.Model)
	if len(stale) > 0 {
		if _, err := UpdatePaths(projectRoot, cfg, stale); err != nil {
			return nil, false
		}
		entries, err = Read(projectRoot)
		if err != nil {
			return nil, false
		}
	}
	entryByPath := map[string]Entry{}
	for _, entry := range entries {
		entryByPath[entry.Path] = entry
	}
	queryTexts := queryEmbeddingTexts(projectRoot, task, cfg)
	taskVectors, err := runEmbeddingCommand(projectRoot, cfg.Embedding.Command, cfg.Embedding.Model, queryTexts)
	if err != nil {
		return nil, false
	}
	type queryVector struct {
		id     string
		part   bool
		vector []float64
	}
	var queryVectors []queryVector
	for _, query := range queryTexts {
		if vec := taskVectors[query.ID]; len(vec) > 0 {
			queryVectors = append(queryVectors, queryVector{id: query.ID, part: query.ID != "__task__", vector: vec})
		}
	}
	if len(queryVectors) == 0 {
		return nil, false
	}
	maxSelected := cfg.MaxSelected
	if maxSelected <= 0 {
		maxSelected = 8
	}
	minScore := cfg.Embedding.MinScore
	if minScore == 0 {
		minScore = defaultMinScore
	}
	scoredByPath := map[string]scoredItem{}
	allByQuery := map[string][]scoredItem{}
	highConfidence := false
	for _, c := range current {
		entry, ok := entryByPath[c.Path]
		if !ok || entry.TextHash != c.TextHash || entry.Model != cfg.Embedding.Model {
			continue
		}
		best := -1.0
		for _, query := range queryVectors {
			score := cosine(query.vector, entry.Vector)
			if score > best {
				best = score
			}
			allByQuery[query.id] = append(allByQuery[query.id], scoredItem{path: c.Path, score: score})
			if score >= minScore {
				highConfidence = true
			}
		}
		if best >= minScore {
			scoredByPath[c.Path] = scoredItem{path: c.Path, score: best}
		}
	}
	if highConfidence {
		for _, query := range queryVectors {
			if !query.part {
				continue
			}
			for _, item := range topKRescue(allByQuery[query.id], cfg.Query) {
				if existing, ok := scoredByPath[item.path]; !ok || item.score > existing.score {
					scoredByPath[item.path] = item
				}
			}
		}
	}
	scoredItems := make([]scoredItem, 0, len(scoredByPath))
	for _, item := range scoredByPath {
		scoredItems = append(scoredItems, item)
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

func queryEmbeddingTexts(projectRoot, task string, cfg config.Brief) []embeddingText {
	queries := decomposeTask(projectRoot, task, cfg.Query)
	texts := make([]embeddingText, 0, len(queries))
	for i, query := range queries {
		id := "__task__"
		if i > 0 {
			id = fmt.Sprintf("__task_part_%d__", i)
		}
		texts = append(texts, embeddingText{ID: id, Text: query})
	}
	return texts
}

func decomposeTask(projectRoot, task string, cfg config.BriefQuery) []string {
	maxParts := cfg.MaxParts
	if maxParts <= 0 {
		maxParts = 4
	}
	if maxParts > 8 {
		maxParts = 8
	}
	queries := []string{strings.TrimSpace(task)}
	mode := strings.ToLower(strings.TrimSpace(cfg.Decomposition))
	switch mode {
	case "command", "model":
		parts, err := runQueryDecompositionCommand(projectRoot, cfg.Command, task, maxParts)
		if err == nil {
			queries = append(queries, parts...)
		}
	case "heuristic":
		queries = append(queries, heuristicTaskParts(task, maxParts)...)
	}
	return uniqueQueries(queries, maxParts+1)
}

func runQueryDecompositionCommand(projectRoot, command, task string, maxParts int) ([]string, error) {
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("brief.query.command is required")
	}
	payload, err := json.Marshal(queryDecompositionRequest{Task: task, MaxParts: maxParts})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = projectRoot
	cmd.Stdin = bytes.NewReader(payload)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var response queryDecompositionResponse
	if err := json.Unmarshal(bytes.TrimSpace(out), &response); err != nil {
		return nil, err
	}
	return uniqueQueries(response.Queries, maxParts), nil
}

func heuristicTaskParts(task string, maxParts int) []string {
	normalized := strings.TrimSpace(task)
	if normalized == "" {
		return nil
	}
	for _, sep := range []string{" и ", " and ", ";", ","} {
		parts := splitQuery(normalized, sep, maxParts)
		if len(parts) > 1 {
			return parts
		}
	}
	return nil
}

func splitQuery(task, sep string, maxParts int) []string {
	raw := strings.Split(task, sep)
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if len([]rune(part)) < 3 {
			continue
		}
		parts = append(parts, part)
		if len(parts) >= maxParts {
			break
		}
	}
	return parts
}

func uniqueQueries(values []string, limit int) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func topKRescue(items []scoredItem, cfg config.BriefQuery) []scoredItem {
	topK := cfg.TopKPerPart
	if topK <= 0 {
		return nil
	}
	floor := cfg.MinPartScore
	if floor == 0 {
		floor = 0.80
	}
	sorted := append([]scoredItem(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].score > sorted[j].score })
	rescued := make([]scoredItem, 0, topK)
	for _, item := range sorted {
		if item.score < floor {
			continue
		}
		rescued = append(rescued, item)
		if len(rescued) >= topK {
			break
		}
	}
	return rescued
}

func Read(projectRoot string) ([]Entry, error) {
	file, err := os.Open(IndexPath(projectRoot))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var entries []Entry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

func DiscoverActive(projectRoot string) []Candidate {
	roots := []struct {
		rel, kind, scope string
		exts             []string
	}{
		{".sima/personal/memory/cards", "memory", "personal", []string{".yaml", ".yml", ".md"}},
		{".sima/personal/skills/active", "skill", "personal", []string{".md"}},
		{".sima/team/memory/cards", "memory", "team", []string{".yaml", ".yml", ".md"}},
		{".sima/team/skills/active", "skill", "team", []string{".md"}},
	}
	var candidates []Candidate
	for _, root := range roots {
		absRoot := filepath.Join(projectRoot, filepath.FromSlash(root.rel))
		_ = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !hasExt(path, root.exts) {
				return nil
			}
			relPath := rel(projectRoot, path)
			if c, ok := candidateForPathWithKind(projectRoot, relPath, root.kind, root.scope); ok {
				candidates = append(candidates, c)
			}
			return nil
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Path < candidates[j].Path })
	return candidates
}

func embedCandidates(projectRoot string, cfg config.Brief, candidates []Candidate) ([]Entry, int, error) {
	if len(candidates) == 0 {
		return nil, 0, nil
	}
	texts := make([]embeddingText, 0, len(candidates))
	for _, c := range candidates {
		texts = append(texts, embeddingText{ID: c.Path, Path: c.Path, Text: c.Text})
	}
	vectors, err := runEmbeddingCommand(projectRoot, cfg.Embedding.Command, cfg.Embedding.Model, texts)
	if err != nil {
		return nil, 0, err
	}
	stamp := time.Now().UTC().Format(time.RFC3339)
	entries := make([]Entry, 0, len(candidates))
	skipped := 0
	for _, c := range candidates {
		vec := vectors[c.Path]
		if len(vec) == 0 {
			skipped++
			continue
		}
		entries = append(entries, Entry{Path: c.Path, Kind: c.Kind, Scope: c.Scope, Model: cfg.Embedding.Model, TextHash: c.TextHash, Vector: vec, UpdatedAt: stamp})
	}
	return entries, skipped, nil
}

func writeEntries(projectRoot string, entries []Entry) error {
	path := IndexPath(projectRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func stalePaths(candidates []Candidate, entries []Entry, model string) []string {
	byPath := map[string]Entry{}
	for _, entry := range entries {
		byPath[entry.Path] = entry
	}
	var stale []string
	for _, c := range candidates {
		entry, ok := byPath[c.Path]
		if !ok || entry.TextHash != c.TextHash || entry.Model != model || len(entry.Vector) == 0 {
			stale = append(stale, c.Path)
		}
	}
	return stale
}

func candidateForPath(projectRoot, relPath string) (Candidate, bool) {
	kind, scope := kindScopeForPath(cleanRel(relPath))
	return candidateForPathWithKind(projectRoot, relPath, kind, scope)
}

func candidateForPathWithKind(projectRoot, relPath, kind, scope string) (Candidate, bool) {
	relPath = cleanRel(relPath)
	if relPath == "" {
		return Candidate{}, false
	}
	abs := filepath.Join(projectRoot, filepath.FromSlash(relPath))
	data, err := os.ReadFile(abs)
	if err != nil || !activeKnowledgeFile(abs, string(data)) {
		return Candidate{}, false
	}
	text := RelevanceText(string(data))
	return Candidate{Path: relPath, Kind: kind, Scope: scope, Text: text, TextHash: hash(text)}, true
}

func kindScopeForPath(path string) (string, string) {
	switch {
	case strings.Contains(path, "/memory/cards/"):
		if strings.Contains(path, "/team/") {
			return "memory", "team"
		}
		return "memory", "personal"
	case strings.Contains(path, "/skills/active/"):
		if strings.Contains(path, "/team/") {
			return "skill", "team"
		}
		return "skill", "personal"
	default:
		return "", ""
	}
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

func RelevanceText(content string) string {
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

func activeKnowledgeFile(path, content string) bool {
	status := statusFromContent(path, content)
	return status == "active"
}

func statusFromContent(path, content string) string {
	if hasExt(path, []string{".yaml", ".yml"}) {
		var meta struct {
			Status string `yaml:"status"`
		}
		if err := yaml.Unmarshal([]byte(content), &meta); err == nil {
			return strings.TrimSpace(meta.Status)
		}
		return ""
	}
	frontmatter := markdownFrontmatter(content)
	if frontmatter == "" {
		return ""
	}
	var meta struct {
		Status string `yaml:"status"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.Status)
}

func markdownFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---\n") {
		return ""
	}
	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return ""
	}
	return rest[:idx]
}

func hasExt(path string, exts []string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, allowed := range exts {
		if ext == allowed {
			return true
		}
	}
	return false
}

func cleanRel(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	path = strings.TrimPrefix(path, "./")
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "../") || path == ".." {
		return ""
	}
	return path
}

func hash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func rel(root, path string) string {
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relPath)
}
