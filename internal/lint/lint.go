package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/antowkos/sima/internal/contracts"
	"github.com/antowkos/sima/internal/proposal"
)

type Issue struct {
	Severity string
	Path     string
	Message  string
}

type Result struct {
	Issues []Issue
}

func (r Result) ErrorCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == "error" {
			count++
		}
	}
	return count
}

func (r Result) WarningCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == "warn" {
			count++
		}
	}
	return count
}

func Check(projectRoot string) (Result, error) {
	simaRoot := filepath.Join(projectRoot, ".sima")
	if _, err := os.Stat(simaRoot); err != nil {
		if os.IsNotExist(err) {
			return Result{Issues: []Issue{{Severity: "error", Path: ".sima", Message: "SIMA state is missing; run sima init"}}}, nil
		}
		return Result{}, err
	}
	var issues []Issue
	stores := []storeSpec{
		{kind: "memory", root: filepath.Join(simaRoot, "personal", "memory", "cards"), exts: []string{".yaml", ".yml", ".md"}},
		{kind: "memory", root: filepath.Join(simaRoot, "team", "memory", "cards"), exts: []string{".yaml", ".yml", ".md"}},
		{kind: "skill", root: filepath.Join(simaRoot, "personal", "skills", "active"), exts: []string{".md"}},
		{kind: "skill", root: filepath.Join(simaRoot, "team", "skills", "active"), exts: []string{".md"}},
	}
	for _, store := range stores {
		issues = append(issues, lintStore(projectRoot, store)...)
	}
	issues = append(issues, lintCandidates(projectRoot, filepath.Join(simaRoot, "personal", "memory", "candidates"))...)
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Severity != issues[j].Severity {
			return issues[i].Severity < issues[j].Severity
		}
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		return issues[i].Message < issues[j].Message
	})
	return Result{Issues: issues}, nil
}

type storeSpec struct {
	kind string
	root string
	exts []string
}

type knowledgeMeta struct {
	ID      string `yaml:"id"`
	Status  string `yaml:"status"`
	Type    string `yaml:"type"`
	Title   string `yaml:"title"`
	Name    string `yaml:"name"`
	Trigger string `yaml:"trigger"`
	Summary string `yaml:"summary"`
}

func lintStore(projectRoot string, store storeSpec) []Issue {
	var issues []Issue
	_ = filepath.WalkDir(store.root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !hasExt(path, store.exts) {
			return nil
		}
		data, err := os.ReadFile(path)
		relPath := rel(projectRoot, path)
		if err != nil {
			issues = append(issues, Issue{Severity: "error", Path: relPath, Message: fmt.Sprintf("read failed: %v", err)})
			return nil
		}
		meta, parseErr := parseKnowledgeMeta(path, string(data))
		if parseErr != nil {
			issues = append(issues, Issue{Severity: "error", Path: relPath, Message: fmt.Sprintf("metadata parse failed: %v", parseErr)})
			return nil
		}
		status := normalizeStatus(meta.Status)
		if status == "" {
			status = "active"
		}
		if !oneOf(status, []string{"active", "deprecated", "superseded", "archived"}) {
			issues = append(issues, Issue{Severity: "error", Path: relPath, Message: "status must be active, deprecated, superseded, or archived"})
		}
		if store.kind == "memory" {
			issues = append(issues, lintMemory(relPath, meta)...)
		} else {
			issues = append(issues, lintSkill(relPath, meta, string(data))...)
		}
		return nil
	})
	return issues
}

func lintMemory(path string, meta knowledgeMeta) []Issue {
	var issues []Issue
	if strings.TrimSpace(meta.Title) == "" {
		issues = append(issues, Issue{Severity: "error", Path: path, Message: "memory title is required"})
	}
	if strings.TrimSpace(meta.Type) != "" && !oneOf(meta.Type, contracts.MemoryTypes) {
		issues = append(issues, Issue{Severity: "error", Path: path, Message: "memory type is unsupported"})
	}
	if normalizeStatus(meta.Status) == "active" || strings.TrimSpace(meta.Status) == "" {
		if strings.TrimSpace(meta.Summary) == "" {
			issues = append(issues, Issue{Severity: "warn", Path: path, Message: "active memory summary is empty"})
		}
		if strings.TrimSpace(meta.Trigger) == "" {
			issues = append(issues, Issue{Severity: "warn", Path: path, Message: "active memory trigger is empty"})
		}
	}
	return issues
}

func lintSkill(path string, meta knowledgeMeta, content string) []Issue {
	var issues []Issue
	name := strings.TrimSpace(meta.Name)
	if name == "" {
		name = firstHeading(content)
	}
	if name == "" {
		issues = append(issues, Issue{Severity: "error", Path: path, Message: "skill name or heading is required"})
	}
	return issues
}

func lintCandidates(projectRoot, candidateDir string) []Issue {
	var issues []Issue
	_ = filepath.WalkDir(candidateDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !hasExt(path, []string{".yaml", ".yml"}) {
			return nil
		}
		relPath := rel(projectRoot, path)
		data, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, Issue{Severity: "error", Path: relPath, Message: fmt.Sprintf("read failed: %v", err)})
			return nil
		}
		var p proposal.Proposal
		if err := yaml.Unmarshal(data, &p); err != nil {
			issues = append(issues, Issue{Severity: "error", Path: relPath, Message: fmt.Sprintf("proposal parse failed: %v", err)})
			return nil
		}
		if p.Status == "candidate" {
			issues = append(issues, Issue{Severity: "warn", Path: relPath, Message: "pending candidate still needs archivist/apply/cleanup"})
		}
		if strings.TrimSpace(p.Learning.Target.Path) != "" && !insideProject(projectRoot, p.Learning.Target.Path) {
			issues = append(issues, Issue{Severity: "error", Path: relPath, Message: "learning.target.path resolves outside project root"})
		}
		return nil
	})
	return issues
}

func parseKnowledgeMeta(path, content string) (knowledgeMeta, error) {
	if hasExt(path, []string{".yaml", ".yml"}) {
		var meta knowledgeMeta
		if err := yaml.Unmarshal([]byte(content), &meta); err != nil {
			return knowledgeMeta{}, err
		}
		return meta, nil
	}
	frontmatter := markdownFrontmatter(content)
	if frontmatter == "" {
		return knowledgeMeta{}, nil
	}
	var meta knowledgeMeta
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return knowledgeMeta{}, err
	}
	return meta, nil
}

func insideProject(projectRoot, relOrAbs string) bool {
	path := relOrAbs
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectRoot, filepath.FromSlash(relOrAbs))
	}
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relPath, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	return relPath != ".." && !strings.HasPrefix(relPath, ".."+string(os.PathSeparator))
}

func markdownFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---\n") {
		return ""
	}
	parts := strings.SplitN(content, "\n---\n", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimPrefix(parts[0], "---\n")
}

func firstHeading(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func normalizeStatus(status string) string {
	return strings.TrimSpace(strings.ToLower(status))
}

func oneOf(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
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
	relPath, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relPath)
}
