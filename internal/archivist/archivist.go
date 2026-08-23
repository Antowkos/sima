package archivist

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
	Decision     string
	Notes        []string
}

func Decide(projectRoot string, opts Options) (Result, error) {
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
	if p.Status != "candidate" {
		return Result{}, fmt.Errorf("proposal status must be candidate, got %q", p.Status)
	}

	decision, notes := decide(projectRoot, proposalPath, p)
	p.ArchivistDecision = decision
	p.ArchivistAt = opts.Now.UTC().Format(time.RFC3339)
	p.ArchivistNotes = notes
	data, err := yaml.Marshal(p)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(proposalPath, data, 0o644); err != nil {
		return Result{}, err
	}
	return Result{ProposalPath: rel(projectRoot, proposalPath), Decision: decision, Notes: notes}, nil
}

func decide(projectRoot, proposalPath string, p proposal.Proposal) (string, []string) {
	item := reviewItem(projectRoot, proposalPath)
	if len(item.Problems) > 0 {
		return "reject", append([]string{"review validation failed"}, item.Problems...)
	}
	if p.Scope != "personal" {
		return "defer", []string{"v0 archivist only auto-approves personal scope"}
	}
	if p.Safety.Decision != "safe" {
		notes := []string{fmt.Sprintf("safety decision is %s", p.Safety.Decision)}
		notes = append(notes, p.Safety.Flags...)
		return "reject", notes
	}
	if len(p.CandidateMemories)+len(p.CandidateSkills) == 0 {
		return "reject", []string{"proposal has no candidate memories or skills"}
	}
	if conflicts := activeConflicts(projectRoot, p); len(conflicts) > 0 {
		return "defer", append([]string{"active output already exists; manual dedup/update needed"}, conflicts...)
	}
	return "apply", []string{"deterministic archivist approved: valid personal safe proposal with evidence and no active output conflict"}
}

func activeConflicts(projectRoot string, p proposal.Proposal) []string {
	var conflicts []string
	for i, c := range p.CandidateMemories {
		path := filepath.Join(projectRoot, ".sima", "personal", "memory", "cards", uniqueID(p.ID, c.Title, i)+".yaml")
		if _, err := os.Stat(path); err == nil {
			conflicts = append(conflicts, rel(projectRoot, path))
		}
	}
	for i, c := range p.CandidateSkills {
		path := filepath.Join(projectRoot, ".sima", "personal", "skills", "active", uniqueID(p.ID, c.Name, i)+".md")
		if _, err := os.Stat(path); err == nil {
			conflicts = append(conflicts, rel(projectRoot, path))
		}
	}
	return conflicts
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

func rel(projectRoot, path string) string {
	relPath, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relPath)
}
