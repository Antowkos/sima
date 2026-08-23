package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/antowkos/sima/internal/proposal"
	"github.com/antowkos/sima/internal/review"
)

type Options struct {
	Target string
	Now    time.Time
}

type Result struct {
	ProposalPath string
	Applied      []string
}

type ActiveMemoryCard struct {
	ID        string              `yaml:"id"`
	Type      string              `yaml:"type"`
	Title     string              `yaml:"title"`
	Trigger   string              `yaml:"trigger"`
	Summary   string              `yaml:"summary"`
	Status    string              `yaml:"status"`
	Scope     string              `yaml:"scope"`
	Evidence  []proposal.Evidence `yaml:"evidence"`
	Source    Source              `yaml:"source"`
	CreatedAt string              `yaml:"created_at"`
	UpdatedAt string              `yaml:"updated_at"`
}

type Source struct {
	ProposalID string `yaml:"proposal_id"`
	Proposal   string `yaml:"proposal"`
	RunID      string `yaml:"run_id"`
}

func Apply(projectRoot string, opts Options) (Result, error) {
	if strings.TrimSpace(opts.Target) == "" {
		return Result{}, fmt.Errorf("proposal target is required")
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	proposalPath, err := resolveProposal(projectRoot, opts.Target)
	if err != nil {
		return Result{}, err
	}
	p, err := readProposal(proposalPath)
	if err != nil {
		return Result{}, err
	}
	item := reviewItem(projectRoot, proposalPath)
	if len(item.Problems) > 0 {
		return Result{}, fmt.Errorf("proposal is invalid: %s", strings.Join(item.Problems, "; "))
	}
	if p.Status != "candidate" {
		return Result{}, fmt.Errorf("proposal status must be candidate, got %q", p.Status)
	}
	if p.Scope != "personal" {
		return Result{}, fmt.Errorf("v0 apply supports personal scope only, got %q", p.Scope)
	}
	if p.Safety.Decision != "safe" {
		return Result{}, fmt.Errorf("proposal safety must be safe, got %q", p.Safety.Decision)
	}
	if p.ArchivistDecision != "apply" {
		return Result{}, fmt.Errorf("proposal archivist_decision must be apply, got %q", p.ArchivistDecision)
	}
	if len(p.CandidateMemories)+len(p.CandidateSkills) == 0 {
		return Result{}, fmt.Errorf("proposal has no candidate memories or skills")
	}

	var applied []string
	stamp := opts.Now.UTC().Format(time.RFC3339)
	for i, c := range p.CandidateMemories {
		path, err := writeMemory(projectRoot, proposalPath, p, c, i, stamp)
		if err != nil {
			return Result{}, err
		}
		applied = append(applied, path)
	}
	for i, c := range p.CandidateSkills {
		path, err := writeSkill(projectRoot, proposalPath, p, c, i, stamp)
		if err != nil {
			return Result{}, err
		}
		applied = append(applied, path)
	}
	p.Status = "applied"
	p.AppliedAt = stamp
	data, err := yaml.Marshal(p)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(proposalPath, data, 0o644); err != nil {
		return Result{}, err
	}
	return Result{ProposalPath: rel(projectRoot, proposalPath), Applied: applied}, nil
}

func writeMemory(projectRoot, proposalPath string, p proposal.Proposal, c proposal.Candidate, index int, stamp string) (string, error) {
	id := uniqueID(p.ID, c.Title, index)
	card := ActiveMemoryCard{
		ID:       id,
		Type:     c.Type,
		Title:    c.Title,
		Trigger:  c.Trigger,
		Summary:  c.Summary,
		Status:   "active",
		Scope:    p.Scope,
		Evidence: c.Evidence,
		Source: Source{
			ProposalID: p.ID,
			Proposal:   rel(projectRoot, proposalPath),
			RunID:      p.Run.ID,
		},
		CreatedAt: stamp,
		UpdatedAt: stamp,
	}
	data, err := yaml.Marshal(card)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(projectRoot, ".sima", "personal", "memory", "cards")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, id+".yaml")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("active memory already exists: %s", rel(projectRoot, path))
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return rel(projectRoot, path), nil
}

func writeSkill(projectRoot, proposalPath string, p proposal.Proposal, c proposal.CandidateSkill, index int, stamp string) (string, error) {
	name := uniqueID(p.ID, c.Name, index)
	content := fmt.Sprintf(`---
name: %s
description: %q
scope: personal
source_proposal: %s
source_run: %s
created_at: %s
updated_at: %s
---

# %s

## Trigger

%s

## Summary

%s

## Evidence

%s
`, name, c.Trigger, rel(projectRoot, proposalPath), p.Run.ID, stamp, stamp, c.Name, c.Trigger, c.Summary, renderEvidence(c.Evidence))
	dir := filepath.Join(projectRoot, ".sima", "personal", "skills", "active")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name+".md")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("active skill already exists: %s", rel(projectRoot, path))
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return rel(projectRoot, path), nil
}

func resolveProposal(projectRoot, target string) (string, error) {
	candidates := []string{target}
	if !filepath.IsAbs(target) {
		candidates = append(candidates,
			filepath.Join(projectRoot, target),
			filepath.Join(projectRoot, ".sima", "personal", "memory", "candidates", target),
			filepath.Join(projectRoot, ".sima", "personal", "memory", "candidates", target+".yaml"),
		)
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("proposal %q not found", target)
}

func readProposal(path string) (proposal.Proposal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return proposal.Proposal{}, err
	}
	var p proposal.Proposal
	if err := yaml.Unmarshal(data, &p); err != nil {
		return proposal.Proposal{}, err
	}
	return p, nil
}

func reviewItem(projectRoot, path string) review.Item {
	result, err := review.Review(projectRoot, review.Options{All: true})
	if err != nil {
		return review.Item{Problems: []string{err.Error()}}
	}
	target := rel(projectRoot, path)
	for _, item := range result.Items {
		if item.Path == target {
			return item
		}
	}
	return review.Item{Problems: []string{"proposal not found in review queue"}}
}

func uniqueID(proposalID, title string, index int) string {
	base := slug(title)
	if base == "" {
		base = slug(proposalID)
	}
	return fmt.Sprintf("%s-%02d-%s", slug(proposalID), index+1, base)
}

func slug(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func renderEvidence(items []proposal.Evidence) string {
	if len(items) == 0 {
		return "- none\n"
	}
	var b strings.Builder
	for _, ev := range items {
		fmt.Fprintf(&b, "- `%s` `%s`", ev.Kind, ev.Path)
		if ev.Note != "" {
			fmt.Fprintf(&b, " — %s", ev.Note)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func rel(projectRoot, path string) string {
	relPath, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relPath)
}
