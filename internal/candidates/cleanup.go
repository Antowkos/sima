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
