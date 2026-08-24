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
	if p.Learning.Destination != "" && !oneOf(p.Learning.Destination, []string{"memory", "skill", "mixed"}) {
		return Result{}, fmt.Errorf("proposal learning.destination must be memory, skill, or mixed, got %q", p.Learning.Destination)
	}
	if p.Learning.Destination != "" && !learningQualityPasses(p.Learning.Quality) {
		return Result{}, fmt.Errorf("proposal learning quality gate did not pass")
	}
	op := learningOperation(p)
	if op != "deprecate" && len(p.CandidateMemories)+len(p.CandidateSkills) == 0 {
		return Result{}, fmt.Errorf("proposal has no candidate memories or skills")
	}

	var applied []string
	stamp := opts.Now.UTC().Format(time.RFC3339)
	switch op {
	case "create":
		paths, err := createOutputs(projectRoot, proposalPath, p, stamp)
		if err != nil {
			return Result{}, err
		}
		applied = append(applied, paths...)
	case "update":
		path, err := updateTarget(projectRoot, proposalPath, p, stamp)
		if err != nil {
			return Result{}, err
		}
		applied = append(applied, path)
	case "supersede":
		path, err := markTargetStatus(projectRoot, p.Learning.Target, "superseded", stamp)
		if err != nil {
			return Result{}, err
		}
		applied = append(applied, path)
		paths, err := createOutputs(projectRoot, proposalPath, p, stamp)
		if err != nil {
			return Result{}, err
		}
		applied = append(applied, paths...)
	case "deprecate":
		path, err := markTargetStatus(projectRoot, p.Learning.Target, "deprecated", stamp)
		if err != nil {
			return Result{}, err
		}
		applied = append(applied, path)
	default:
		return Result{}, fmt.Errorf("unsupported learning.operation %q", op)
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

func learningOperation(p proposal.Proposal) string {
	if p.Learning.Operation != "" {
		return p.Learning.Operation
	}
	if p.Operation != "" {
		return p.Operation
	}
	return "create"
}

func createOutputs(projectRoot, proposalPath string, p proposal.Proposal, stamp string) ([]string, error) {
	var applied []string
	for i, c := range p.CandidateMemories {
		path, err := writeMemory(projectRoot, proposalPath, p, c, i, stamp)
		if err != nil {
			return nil, err
		}
		applied = append(applied, path)
	}
	for i, c := range p.CandidateSkills {
		path, err := writeSkill(projectRoot, proposalPath, p, c, i, stamp)
		if err != nil {
			return nil, err
		}
		applied = append(applied, path)
	}
	return applied, nil
}

func updateTarget(projectRoot, proposalPath string, p proposal.Proposal, stamp string) (string, error) {
	switch p.Learning.Target.Kind {
	case "memory":
		if len(p.CandidateMemories) != 1 || len(p.CandidateSkills) != 0 {
			return "", fmt.Errorf("memory update requires exactly one candidate memory")
		}
		return updateMemory(projectRoot, proposalPath, p, p.CandidateMemories[0], stamp)
	case "skill":
		if len(p.CandidateSkills) != 1 || len(p.CandidateMemories) != 0 {
			return "", fmt.Errorf("skill update requires exactly one candidate skill")
		}
		return updateSkill(projectRoot, proposalPath, p, p.CandidateSkills[0], stamp)
	default:
		return "", fmt.Errorf("learning.target.kind must be memory or skill")
	}
}

func updateMemory(projectRoot, proposalPath string, p proposal.Proposal, c proposal.Candidate, stamp string) (string, error) {
	path, err := resolveTargetPath(projectRoot, p.Learning.Target)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var card ActiveMemoryCard
	if err := yaml.Unmarshal(data, &card); err != nil {
		return "", err
	}
	if strings.TrimSpace(card.ID) == "" {
		return "", fmt.Errorf("target memory card missing id: %s", rel(projectRoot, path))
	}
	card.Type = c.Type
	card.Title = c.Title
	card.Trigger = c.Trigger
	card.Summary = c.Summary
	card.Status = "active"
	card.Scope = p.Scope
	card.Evidence = c.Evidence
	card.Source = Source{ProposalID: p.ID, Proposal: rel(projectRoot, proposalPath), RunID: p.Run.ID}
	card.UpdatedAt = stamp
	if card.CreatedAt == "" {
		card.CreatedAt = stamp
	}
	updated, err := yaml.Marshal(card)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		return "", err
	}
	return rel(projectRoot, path), nil
}

func updateSkill(projectRoot, proposalPath string, p proposal.Proposal, c proposal.CandidateSkill, stamp string) (string, error) {
	path, err := resolveTargetPath(projectRoot, p.Learning.Target)
	if err != nil {
		return "", err
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	content := skillContent(name, rel(projectRoot, proposalPath), p, c, stamp, "active")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return rel(projectRoot, path), nil
}

func markTargetStatus(projectRoot string, target proposal.LearningTarget, status string, stamp string) (string, error) {
	path, err := resolveTargetPath(projectRoot, target)
	if err != nil {
		return "", err
	}
	switch target.Kind {
	case "memory":
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		var card ActiveMemoryCard
		if err := yaml.Unmarshal(data, &card); err != nil {
			return "", err
		}
		card.Status = status
		card.UpdatedAt = stamp
		updated, err := yaml.Marshal(card)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, updated, 0o644); err != nil {
			return "", err
		}
	case "skill":
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		updated := setSkillFrontmatterStatus(string(data), status, stamp)
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("learning.target.kind must be memory or skill")
	}
	return rel(projectRoot, path), nil
}

func resolveTargetPath(projectRoot string, target proposal.LearningTarget) (string, error) {
	if strings.TrimSpace(target.Path) == "" {
		return "", fmt.Errorf("learning.target.path is required")
	}
	path := target.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectRoot, path)
	}
	cleanRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", err
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	relPath, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		return "", fmt.Errorf("learning.target.path must stay under project root")
	}
	if info, err := os.Stat(cleanPath); err != nil || info.IsDir() {
		return "", fmt.Errorf("learning target not found: %s", target.Path)
	}
	return cleanPath, nil
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
	content := skillContent(name, rel(projectRoot, proposalPath), p, c, stamp, "active")
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

func skillContent(name, sourceProposal string, p proposal.Proposal, c proposal.CandidateSkill, stamp string, status string) string {
	return fmt.Sprintf(`---
name: %s
description: %q
scope: personal
status: %s
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
`, name, c.Trigger, status, sourceProposal, p.Run.ID, stamp, stamp, c.Name, c.Trigger, c.Summary, renderEvidence(c.Evidence))
}

func setSkillFrontmatterStatus(text, status, stamp string) string {
	if !strings.HasPrefix(text, "---\n") {
		return fmt.Sprintf("---\nstatus: %s\nupdated_at: %s\n---\n\n%s", status, stamp, text)
	}
	parts := strings.SplitN(text, "\n---\n", 2)
	if len(parts) != 2 {
		return fmt.Sprintf("---\nstatus: %s\nupdated_at: %s\n---\n\n%s", status, stamp, text)
	}
	lines := strings.Split(parts[0], "\n")
	statusSet := false
	updatedSet := false
	for i, line := range lines {
		if strings.HasPrefix(line, "status:") {
			lines[i] = "status: " + status
			statusSet = true
		}
		if strings.HasPrefix(line, "updated_at:") {
			lines[i] = "updated_at: " + stamp
			updatedSet = true
		}
	}
	if !statusSet {
		lines = append(lines, "status: "+status)
	}
	if !updatedSet {
		lines = append(lines, "updated_at: "+stamp)
	}
	return strings.Join(lines, "\n") + "\n---\n" + parts[1]
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

func learningQualityPasses(q proposal.LearningQuality) bool {
	return q.Durable && q.Triggerable && q.EvidenceBacked && q.NonTransient && q.Reusable
}

func oneOf(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
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
