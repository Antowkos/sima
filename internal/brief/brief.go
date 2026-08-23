package brief

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	briefDir := filepath.Join(simaRoot, "personal", "briefs")
	if err := os.MkdirAll(briefDir, 0o755); err != nil {
		return Result{}, err
	}

	data := data{
		Task:                   opts.Task,
		GeneratedAt:            opts.Now.UTC().Format(time.RFC3339),
		SystemSkills:           listFiles(filepath.Join(simaRoot, "system", "skills"), projectRoot, []string{".md"}),
		PersonalMemory:         listFiles(filepath.Join(simaRoot, "personal", "memory", "cards"), projectRoot, []string{".yaml", ".yml", ".md"}),
		PersonalMemorySnippets: readSnippets(filepath.Join(simaRoot, "personal", "memory", "cards"), projectRoot, []string{".yaml", ".yml", ".md"}),
		PersonalSkills:         listFiles(filepath.Join(simaRoot, "personal", "skills", "active"), projectRoot, []string{".md"}),
		PersonalSkillSnippets:  readSnippets(filepath.Join(simaRoot, "personal", "skills", "active"), projectRoot, []string{".md"}),
		TeamMemory:             listFiles(filepath.Join(simaRoot, "team", "memory", "cards"), projectRoot, []string{".yaml", ".yml", ".md"}),
		TeamMemorySnippets:     readSnippets(filepath.Join(simaRoot, "team", "memory", "cards"), projectRoot, []string{".yaml", ".yml", ".md"}),
		TeamSkills:             listFiles(filepath.Join(simaRoot, "team", "skills", "active"), projectRoot, []string{".md"}),
		TeamSkillSnippets:      readSnippets(filepath.Join(simaRoot, "team", "skills", "active"), projectRoot, []string{".md"}),
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
