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
	if decision == "reject" {
		p.Status = "rejected"
	} else if decision == "defer" && p.Learning.Destination == "session_only" {
		p.Status = "session_only"
	} else if decision == "defer" {
		p.Status = "deferred"
	}
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
	if p.CandidateSource == "structured_invalid" {
		notes := []string{"structured worker proposal is malformed or incomplete"}
		notes = append(notes, p.CandidateErrors...)
		return "defer", notes
	}
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
	if learningOperation(p) != "deprecate" && len(p.CandidateMemories)+len(p.CandidateSkills) == 0 {
		return "reject", []string{"proposal has no candidate memories or skills"}
	}
	if p.CandidateSource == "fallback" {
		return "defer", []string{"fallback review candidates stay session_only until a structured worker proposal exists"}
	}
	if decision, notes, ok := decideLearning(p); ok {
		return decision, notes
	}
	if learningOperation(p) == "create" {
		if conflicts := activeConflicts(projectRoot, p); len(conflicts) > 0 {
			return "defer", append([]string{"active output already exists; manual dedup/update needed"}, conflicts...)
		}
	}
	return "apply", []string{"deterministic archivist approved: valid personal safe proposal with evidence and learning gates passed"}
}

func decideLearning(p proposal.Proposal) (string, []string, bool) {
	if p.Learning.Destination == "" {
		return "", nil, false
	}
	switch p.Learning.Destination {
	case "session_only":
		return "defer", []string{"learning destination is session_only; keep artifacts searchable without promoting active knowledge"}, true
	case "reject":
		return "reject", []string{"learning destination is reject"}, true
	case "memory", "skill", "mixed":
		var failures []string
		q := p.Learning.Quality
		if !q.Durable {
			failures = append(failures, "learning quality failed: durable=false")
		}
		if !q.Triggerable {
			failures = append(failures, "learning quality failed: triggerable=false")
		}
		if !q.EvidenceBacked {
			failures = append(failures, "learning quality failed: evidence_backed=false")
		}
		if !q.NonTransient {
			failures = append(failures, "learning quality failed: non_transient=false")
		}
		if !q.Reusable {
			failures = append(failures, "learning quality failed: reusable=false")
		}
		if len(failures) > 0 {
			return "defer", append([]string{"learning quality gate did not pass"}, append(failures, p.Learning.Notes...)...), true
		}
		return "", nil, false
	default:
		return "reject", []string{fmt.Sprintf("unsupported learning destination %q", p.Learning.Destination)}, true
	}
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

func activeConflicts(projectRoot string, p proposal.Proposal) []string {
	var conflicts []string
	activeMemory := readActiveMemory(projectRoot)
	activeSkills := readActiveSkills(projectRoot)
	for i, c := range p.CandidateMemories {
		path := filepath.Join(projectRoot, ".sima", "personal", "memory", "cards", uniqueID(p.ID, c.Title, i)+".yaml")
		if _, err := os.Stat(path); err == nil {
			conflicts = append(conflicts, rel(projectRoot, path))
		}
		candidateTitle := slug(c.Title)
		candidateTrigger := slug(c.Trigger)
		for _, active := range activeMemory {
			if candidateTitle != "" && candidateTitle == active.TitleSlug {
				conflicts = append(conflicts, fmt.Sprintf("similar active memory title exists: %s", active.Path))
				continue
			}
			if candidateTrigger != "" && candidateTrigger == active.TriggerSlug {
				conflicts = append(conflicts, fmt.Sprintf("similar active memory trigger exists: %s", active.Path))
			}
		}
	}
	for i, c := range p.CandidateSkills {
		path := filepath.Join(projectRoot, ".sima", "personal", "skills", "active", uniqueID(p.ID, c.Name, i)+".md")
		if _, err := os.Stat(path); err == nil {
			conflicts = append(conflicts, rel(projectRoot, path))
		}
		candidateName := slug(c.Name)
		candidateTrigger := slug(c.Trigger)
		for _, active := range activeSkills {
			if candidateName != "" && candidateName == active.NameSlug {
				conflicts = append(conflicts, fmt.Sprintf("similar active skill name exists: %s", active.Path))
				continue
			}
			if candidateTrigger != "" && candidateTrigger == active.TriggerSlug {
				conflicts = append(conflicts, fmt.Sprintf("similar active skill trigger exists: %s", active.Path))
			}
		}
	}
	return conflicts
}

type activeMemoryRef struct {
	Path        string
	TitleSlug   string
	TriggerSlug string
}

type activeSkillRef struct {
	Path        string
	NameSlug    string
	TriggerSlug string
}

func readActiveMemory(projectRoot string) []activeMemoryRef {
	dir := filepath.Join(projectRoot, ".sima", "personal", "memory", "cards")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var refs []activeMemoryRef
	for _, entry := range entries {
		if entry.IsDir() || !isYAML(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var card struct {
			Title   string `yaml:"title"`
			Trigger string `yaml:"trigger"`
			Status  string `yaml:"status"`
		}
		if err := yaml.Unmarshal(data, &card); err != nil || card.Status == "archived" || card.Status == "deprecated" || card.Status == "superseded" {
			continue
		}
		refs = append(refs, activeMemoryRef{Path: rel(projectRoot, path), TitleSlug: slug(card.Title), TriggerSlug: slug(card.Trigger)})
	}
	return refs
}

func readActiveSkills(projectRoot string) []activeSkillRef {
	dir := filepath.Join(projectRoot, ".sima", "personal", "skills", "active")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var refs []activeSkillRef
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		refs = append(refs, activeSkillRef{Path: rel(projectRoot, path), NameSlug: slug(name), TriggerSlug: slug(extractSkillTrigger(string(data)))})
	}
	return refs
}

func extractSkillTrigger(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "## Trigger" {
			for _, candidate := range lines[i+1:] {
				candidate = strings.TrimSpace(candidate)
				if candidate != "" && !strings.HasPrefix(candidate, "#") {
					return candidate
				}
			}
		}
	}
	return ""
}

func isYAML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
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
