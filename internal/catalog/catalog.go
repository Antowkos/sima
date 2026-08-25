package catalog

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Item is a lifecycle-aware knowledge record stored in project-local SIMA state.
type Item struct {
	Status string
	Scope  string
	Kind   string
	Title  string
	Path   string
}

// Options controls lifecycle catalog listing.
type Options struct {
	Kind   string
	Status string
}

// List returns memory or skill items from personal/team active stores.
func List(projectRoot string, opts Options) ([]Item, error) {
	kind := strings.TrimSpace(opts.Kind)
	status := normalizeStatus(opts.Status)
	if status == "" {
		status = "active"
	}

	var roots []rootSpec
	switch kind {
	case "memory":
		roots = []rootSpec{
			{scope: "personal", kind: "memory", path: filepath.Join(projectRoot, ".sima", "personal", "memory", "cards"), exts: []string{".yaml", ".yml", ".md"}},
			{scope: "team", kind: "memory", path: filepath.Join(projectRoot, ".sima", "team", "memory", "cards"), exts: []string{".yaml", ".yml", ".md"}},
		}
	case "skill":
		roots = []rootSpec{
			{scope: "personal", kind: "skill", path: filepath.Join(projectRoot, ".sima", "personal", "skills", "active"), exts: []string{".md"}},
			{scope: "team", kind: "skill", path: filepath.Join(projectRoot, ".sima", "team", "skills", "active"), exts: []string{".md"}},
		}
	default:
		return nil, nil
	}

	var items []Item
	for _, root := range roots {
		items = append(items, listRoot(projectRoot, root, status)...)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return items[i].Status < items[j].Status
		}
		if items[i].Scope != items[j].Scope {
			return items[i].Scope < items[j].Scope
		}
		return items[i].Path < items[j].Path
	})
	return items, nil
}

type rootSpec struct {
	scope string
	kind  string
	path  string
	exts  []string
}

func listRoot(projectRoot string, root rootSpec, statusFilter string) []Item {
	var items []Item
	_ = filepath.WalkDir(root.path, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !hasExt(path, root.exts) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		meta := parseMeta(path, string(data))
		status := normalizeStatus(meta.Status)
		if status == "" {
			status = "active"
		}
		if statusFilter != "all" && status != statusFilter {
			return nil
		}
		title := strings.TrimSpace(meta.Title)
		if title == "" {
			title = strings.TrimSpace(meta.Name)
		}
		if title == "" && root.kind == "skill" {
			title = firstHeading(string(data))
		}
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		items = append(items, Item{Status: status, Scope: root.scope, Kind: root.kind, Title: title, Path: rel(projectRoot, path)})
		return nil
	})
	return items
}

type metadata struct {
	Status string `yaml:"status"`
	Title  string `yaml:"title"`
	Name   string `yaml:"name"`
}

func parseMeta(path, content string) metadata {
	if hasExt(path, []string{".yaml", ".yml"}) {
		var meta metadata
		_ = yaml.Unmarshal([]byte(content), &meta)
		return meta
	}
	frontmatter := markdownFrontmatter(content)
	if frontmatter == "" {
		return metadata{}
	}
	var meta metadata
	_ = yaml.Unmarshal([]byte(frontmatter), &meta)
	return meta
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
