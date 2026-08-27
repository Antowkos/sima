package proposal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type RememberOptions struct {
	Text    string
	Source  string
	Type    string
	Title   string
	Trigger string
	Summary string
	Now     time.Time
}

type RememberResult struct {
	Path       string
	ID         string
	Candidates int
	Source     string
	Safety     string
}

func Remember(projectRoot string, opts RememberOptions) (RememberResult, error) {
	text := strings.TrimSpace(opts.Text)
	if text == "" {
		return RememberResult{}, fmt.Errorf("knowledge text is required")
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	source := strings.TrimSpace(strings.ToLower(opts.Source))
	if source == "" {
		source = "user"
	}
	if source != "user" && source != "review" && source != "agent" {
		return RememberResult{}, fmt.Errorf("source must be user, review, or agent")
	}
	memoryType := strings.TrimSpace(strings.ToLower(opts.Type))
	if memoryType == "" {
		memoryType = "invariant"
	}
	if !validRememberMemoryType(memoryType) {
		return RememberResult{}, fmt.Errorf("type must be decision, invariant, gotcha, workflow, guardrail, anti_pattern, or open_question")
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = titleFromText(text)
	}
	trigger := strings.TrimSpace(opts.Trigger)
	if trigger == "" {
		trigger = fmt.Sprintf("When working in this project and the %s-provided knowledge is relevant.", source)
	}
	summary := strings.TrimSpace(opts.Summary)
	if summary == "" {
		summary = text
	}

	id := fmt.Sprintf("remember-%s-%s", opts.Now.UTC().Format("20060102T150405Z"), slug(title))
	if len(id) > 96 {
		id = id[:96]
		id = strings.TrimRight(id, "-")
	}
	evidencePath, err := writeRememberEvidence(projectRoot, id, source, text, opts.Now)
	if err != nil {
		return RememberResult{}, err
	}
	evidence := []Evidence{{Kind: "direct_knowledge", Path: evidencePath, Note: "explicit knowledge routed through SIMA harness"}}
	proposal := Proposal{
		Version:           1,
		ID:                id,
		Kind:              "direct_knowledge",
		Scope:             "personal",
		Operation:         "create",
		Status:            "candidate",
		ArchivistDecision: "defer",
		Safety:            Safety{Decision: "safe"},
		Run: RunRef{
			ID:     id,
			Path:   evidencePath,
			Status: "success",
		},
		Summary:         fmt.Sprintf("Direct %s-provided knowledge captured for SIMA review.", source),
		CandidateSource: "structured",
		CandidateMemories: []Candidate{{
			Type:     memoryType,
			Title:    title,
			Trigger:  trigger,
			Summary:  summary,
			Evidence: evidence,
		}},
		Evidence:  evidence,
		CreatedAt: opts.Now.UTC().Format(time.RFC3339),
		ReviewInstructions: []string{
			"Review in a clean archivist session before applying.",
			"This came from an explicit remember request; route it through SIMA, not the agent's native memory.",
			"Apply only if it is durable, triggerable, evidence-backed, non-transient project knowledge.",
		},
	}
	proposal.Learning = classifyLearning(projectRoot, proposal)
	candidateDir := filepath.Join(projectRoot, ".sima", "personal", "memory", "candidates")
	if err := os.MkdirAll(candidateDir, 0o755); err != nil {
		return RememberResult{}, err
	}
	path := filepath.Join(candidateDir, id+".yaml")
	data, err := yaml.Marshal(proposal)
	if err != nil {
		return RememberResult{}, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return RememberResult{}, err
	}
	return RememberResult{Path: path, ID: id, Candidates: 1, Source: proposal.CandidateSource, Safety: proposal.Safety.Decision}, nil
}

func writeRememberEvidence(projectRoot, id, source, text string, now time.Time) (string, error) {
	dir := filepath.Join(projectRoot, ".sima", "personal", "evidence", "direct")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, id+".md")
	content := fmt.Sprintf("# Direct SIMA Knowledge\n\nsource: %s\ncreated_at: %s\n\n## Knowledge\n\n%s\n", source, now.UTC().Format(time.RFC3339), text)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return rel(projectRoot, path), nil
}

func titleFromText(text string) string {
	clean := strings.Join(strings.Fields(text), " ")
	if len(clean) > 64 {
		clean = clean[:64]
		if idx := strings.LastIndex(clean, " "); idx > 20 {
			clean = clean[:idx]
		}
	}
	clean = strings.Trim(clean, " .,:;\t\n\r")
	if clean == "" {
		return "Direct project knowledge"
	}
	return clean
}

func validRememberMemoryType(value string) bool {
	switch value {
	case "decision", "invariant", "gotcha", "workflow", "guardrail", "anti_pattern", "open_question":
		return true
	default:
		return false
	}
}
