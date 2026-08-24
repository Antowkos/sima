package review

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/antowkos/sima/internal/proposal"
)

type Options struct {
	All bool
}

type Result struct {
	Items []Item
}

type Item struct {
	Path       string
	ID         string
	RunID      string
	Status     string
	Decision   string
	Safety     string
	Candidates int
	Evidence   int
	Problems   []string
}

func Review(projectRoot string, opts Options) (Result, error) {
	candidateDir := filepath.Join(projectRoot, ".sima", "personal", "memory", "candidates")
	entries, err := os.ReadDir(candidateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, nil
		}
		return Result{}, err
	}
	var items []Item
	for _, entry := range entries {
		if entry.IsDir() || !isYAML(entry.Name()) {
			continue
		}
		path := filepath.Join(candidateDir, entry.Name())
		item := readProposal(projectRoot, path)
		if !opts.All && item.Status != "candidate" {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return Result{Items: items}, nil
}

func readProposal(projectRoot, path string) Item {
	item := Item{Path: rel(projectRoot, path)}
	data, err := os.ReadFile(path)
	if err != nil {
		item.Problems = append(item.Problems, fmt.Sprintf("read failed: %v", err))
		return item
	}
	var p proposal.Proposal
	if err := yaml.Unmarshal(data, &p); err != nil {
		item.Problems = append(item.Problems, fmt.Sprintf("parse failed: %v", err))
		return item
	}
	item.ID = p.ID
	item.RunID = p.Run.ID
	item.Status = p.Status
	item.Decision = p.ArchivistDecision
	item.Safety = p.Safety.Decision
	item.Candidates = len(p.CandidateMemories) + len(p.CandidateSkills)
	item.Evidence = len(p.Evidence)
	item.Problems = validate(p)
	return item
}

func validate(p proposal.Proposal) []string {
	var problems []string
	if p.Version == 0 {
		problems = append(problems, "missing version")
	}
	if strings.TrimSpace(p.ID) == "" {
		problems = append(problems, "missing id")
	}
	if strings.TrimSpace(p.Kind) == "" {
		problems = append(problems, "missing kind")
	}
	if p.Scope != "personal" && p.Scope != "team" {
		problems = append(problems, "scope must be personal or team")
	}
	if !oneOf(p.Operation, []string{"create", "update", "deprecate", "supersede"}) {
		problems = append(problems, "unsupported operation")
	}
	if !oneOf(p.Status, []string{"candidate", "applied", "rejected", "deferred"}) {
		problems = append(problems, "unsupported status")
	}
	if !oneOf(p.ArchivistDecision, []string{"apply", "reject", "defer"}) {
		problems = append(problems, "archivist_decision must be apply, reject, or defer")
	}
	if !oneOf(p.Safety.Decision, []string{"safe", "suspicious", "unsafe"}) {
		problems = append(problems, "safety.decision must be safe, suspicious, or unsafe")
	}
	if p.Run.ID == "" || p.Run.Path == "" {
		problems = append(problems, "missing run reference")
	}
	if len(p.Evidence) == 0 {
		problems = append(problems, "missing evidence")
	}
	if p.CandidateSource != "" && !oneOf(p.CandidateSource, []string{"structured", "structured_invalid", "fallback"}) {
		problems = append(problems, "candidate_source must be structured, structured_invalid, or fallback")
	}
	for _, problem := range p.CandidateErrors {
		problems = append(problems, "candidate error: "+problem)
	}
	for i, ev := range p.Evidence {
		if strings.TrimSpace(ev.Kind) == "" || strings.TrimSpace(ev.Path) == "" {
			problems = append(problems, fmt.Sprintf("evidence[%d] requires kind and path", i))
		}
	}
	for i, c := range p.CandidateMemories {
		if strings.TrimSpace(c.Type) == "" || strings.TrimSpace(c.Title) == "" || strings.TrimSpace(c.Trigger) == "" || strings.TrimSpace(c.Summary) == "" {
			problems = append(problems, fmt.Sprintf("candidate_memories[%d] requires type, title, trigger, and summary", i))
		}
		if len(c.Evidence) == 0 {
			problems = append(problems, fmt.Sprintf("candidate_memories[%d] missing evidence", i))
		}
	}
	for i, c := range p.CandidateSkills {
		if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Trigger) == "" || strings.TrimSpace(c.Summary) == "" {
			problems = append(problems, fmt.Sprintf("candidate_skills[%d] requires name, trigger, and summary", i))
		}
		if len(c.Evidence) == 0 {
			problems = append(problems, fmt.Sprintf("candidate_skills[%d] missing evidence", i))
		}
	}
	if p.Safety.Decision != "safe" && p.ArchivistDecision == "apply" {
		problems = append(problems, "suspicious/unsafe proposals cannot be apply")
	}
	return problems
}

func oneOf(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
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
