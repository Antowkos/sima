package candidates

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/antowkos/sima/internal/proposal"
)

type CleanupOptions struct {
	Now time.Time
}

type CleanupResult struct {
	Updated []string
}

type ListOptions struct {
	Status string
}

type Item struct {
	ID          string
	Status      string
	Decision    string
	Safety      string
	Destination string
	Operation   string
	Candidates  int
	Path        string
}

type ShowResult struct {
	Path     string
	Content  string
	Proposal proposal.Proposal
}

func List(projectRoot string, opts ListOptions) ([]Item, error) {
	statusFilter := strings.TrimSpace(strings.ToLower(opts.Status))
	if statusFilter == "" {
		statusFilter = "candidate"
	}
	candidateDir := filepath.Join(projectRoot, ".sima", "personal", "memory", "candidates")
	entries, err := os.ReadDir(candidateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var items []Item
	for _, entry := range entries {
		if entry.IsDir() || !isYAML(entry.Name()) {
			continue
		}
		path := filepath.Join(candidateDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var p proposal.Proposal
		if err := yaml.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		status := strings.TrimSpace(strings.ToLower(p.Status))
		if statusFilter != "all" && status != statusFilter {
			continue
		}
		items = append(items, Item{
			ID:          p.ID,
			Status:      p.Status,
			Decision:    p.ArchivistDecision,
			Safety:      p.Safety.Decision,
			Destination: p.Learning.Destination,
			Operation:   p.Learning.Operation,
			Candidates:  len(p.CandidateMemories) + len(p.CandidateSkills),
			Path:        rel(projectRoot, path),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return items, nil
}

func Show(projectRoot, target string) (ShowResult, error) {
	path, err := resolveCandidate(projectRoot, target)
	if err != nil {
		return ShowResult{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ShowResult{}, err
	}
	var p proposal.Proposal
	if err := yaml.Unmarshal(data, &p); err != nil {
		return ShowResult{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return ShowResult{Path: rel(projectRoot, path), Content: string(data), Proposal: p}, nil
}

// CleanupDeferred marks deferred pending proposals as no longer pending.
func CleanupDeferred(projectRoot string, opts CleanupOptions) (CleanupResult, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	candidateDir := filepath.Join(projectRoot, ".sima", "personal", "memory", "candidates")
	entries, err := os.ReadDir(candidateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return CleanupResult{}, nil
		}
		return CleanupResult{}, err
	}
	var updated []string
	for _, entry := range entries {
		if entry.IsDir() || !isYAML(entry.Name()) {
			continue
		}
		path := filepath.Join(candidateDir, entry.Name())
		changed, err := cleanupFile(path, opts.Now)
		if err != nil {
			return CleanupResult{}, err
		}
		if changed {
			updated = append(updated, rel(projectRoot, path))
		}
	}
	sort.Strings(updated)
	return CleanupResult{Updated: updated}, nil
}

func cleanupFile(path string, now time.Time) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var p proposal.Proposal
	if err := yaml.Unmarshal(data, &p); err != nil {
		return false, fmt.Errorf("parse %s: %w", path, err)
	}
	if p.Status != "candidate" || p.ArchivistDecision != "defer" {
		return false, nil
	}
	text := string(data)
	text = replaceTopLevelField(text, "status", "deferred")
	if !strings.Contains(text, "cleanup_at:") {
		text += fmt.Sprintf("cleanup_at: %q\n", now.UTC().Format(time.RFC3339))
	}
	if !strings.Contains(text, "cleanup_note:") {
		text += "cleanup_note: deferred pending candidate cleaned from active review queue\n"
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func replaceTopLevelField(text, key, value string) string {
	prefix := key + ":"
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = prefix + " " + value
			return strings.Join(lines, "\n")
		}
	}
	if strings.HasSuffix(text, "\n") {
		return text + prefix + " " + value + "\n"
	}
	return text + "\n" + prefix + " " + value
}

func resolveCandidate(projectRoot, target string) (string, error) {
	if strings.TrimSpace(target) == "" {
		return "", fmt.Errorf("candidate id or path is required")
	}
	candidateDir := filepath.Join(projectRoot, ".sima", "personal", "memory", "candidates")
	candidates := []string{target}
	if !filepath.IsAbs(target) {
		candidates = append(candidates,
			filepath.Join(projectRoot, target),
			filepath.Join(candidateDir, target),
			filepath.Join(candidateDir, target+".yaml"),
		)
	}
	candidateRoot, err := filepath.Abs(candidateDir)
	if err != nil {
		return "", err
	}
	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if !within(abs, candidateRoot) {
			continue
		}
		info, err := os.Stat(abs)
		if err == nil && !info.IsDir() {
			return abs, nil
		}
	}
	return "", fmt.Errorf("candidate not found: %s", target)
}

func within(path, root string) bool {
	if path == root {
		return true
	}
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relPath != ".." && !strings.HasPrefix(relPath, ".."+string(filepath.Separator))
}

func isYAML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}

func rel(projectRoot, path string) string {
	relPath, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relPath)
}
