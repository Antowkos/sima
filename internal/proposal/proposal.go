package proposal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Options struct {
	FromRun string
	Now     time.Time
}

type Result struct {
	Path       string
	RunID      string
	Decision   string
	Safety     string
	Candidates int
	Source     string
}

type WorkerReport struct {
	RunID          string           `yaml:"run_id" json:"run_id,omitempty"`
	Backend        string           `yaml:"backend" json:"backend,omitempty"`
	Status         string           `yaml:"status" json:"status,omitempty"`
	Task           string           `yaml:"task" json:"task,omitempty"`
	BriefPath      string           `yaml:"brief_path" json:"brief_path,omitempty"`
	ExitCode       int              `yaml:"exit_code" json:"exit_code,omitempty"`
	StdoutPath     string           `yaml:"stdout_path" json:"stdout_path,omitempty"`
	StderrPath     string           `yaml:"stderr_path" json:"stderr_path,omitempty"`
	ProposedMemory []Candidate      `yaml:"proposed_memory,omitempty" json:"proposed_memory,omitempty"`
	ProposedSkills []CandidateSkill `yaml:"proposed_skills,omitempty" json:"proposed_skills,omitempty"`
}

type Proposal struct {
	Version            int              `yaml:"version"`
	ID                 string           `yaml:"id"`
	Kind               string           `yaml:"kind"`
	Scope              string           `yaml:"scope"`
	Operation          string           `yaml:"operation"`
	Status             string           `yaml:"status"`
	ArchivistDecision  string           `yaml:"archivist_decision"`
	Safety             Safety           `yaml:"safety"`
	Run                RunRef           `yaml:"run"`
	Summary            string           `yaml:"summary"`
	CandidateMemories  []Candidate      `yaml:"candidate_memories,omitempty"`
	CandidateSkills    []CandidateSkill `yaml:"candidate_skills,omitempty"`
	CandidateSource    string           `yaml:"candidate_source,omitempty"`
	CandidateErrors    []string         `yaml:"candidate_errors,omitempty"`
	Evidence           []Evidence       `yaml:"evidence"`
	CreatedAt          string           `yaml:"created_at"`
	AppliedAt          string           `yaml:"applied_at,omitempty"`
	ArchivistAt        string           `yaml:"archivist_at,omitempty"`
	ArchivistNotes     []string         `yaml:"archivist_notes,omitempty"`
	ReviewInstructions []string         `yaml:"review_instructions"`
}

type Safety struct {
	Decision string   `yaml:"decision"`
	Flags    []string `yaml:"flags,omitempty"`
}

type RunRef struct {
	ID     string `yaml:"id"`
	Path   string `yaml:"path"`
	Status string `yaml:"status"`
}

type Candidate struct {
	Type     string     `yaml:"type" json:"type"`
	Title    string     `yaml:"title" json:"title"`
	Trigger  string     `yaml:"trigger" json:"trigger"`
	Summary  string     `yaml:"summary" json:"summary"`
	Evidence []Evidence `yaml:"evidence" json:"evidence,omitempty"`
}

type CandidateSkill struct {
	Name     string     `yaml:"name" json:"name"`
	Trigger  string     `yaml:"trigger" json:"trigger"`
	Summary  string     `yaml:"summary" json:"summary"`
	Evidence []Evidence `yaml:"evidence" json:"evidence,omitempty"`
}

type Evidence struct {
	Kind string `yaml:"kind" json:"kind"`
	Path string `yaml:"path" json:"path"`
	Note string `yaml:"note,omitempty" json:"note,omitempty"`
}

func Generate(projectRoot string, opts Options) (Result, error) {
	if strings.TrimSpace(opts.FromRun) == "" {
		return Result{}, fmt.Errorf("--from-run is required")
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	runDir, runID, err := resolveRun(projectRoot, opts.FromRun)
	if err != nil {
		return Result{}, err
	}
	report, err := readWorkerReport(runDir)
	if err != nil {
		return Result{}, err
	}
	if report.RunID != "" {
		runID = report.RunID
	}

	stdoutText := readText(filepath.Join(runDir, "stdout.log"), 4096)
	stderrText := readText(filepath.Join(runDir, "stderr.log"), 4096)
	stdoutReport, stdoutParse := parseWorkerOutput(stdoutText)
	safety := assessSafety(stdoutText + "\n" + stderrText)
	summary := summarize(report, stdoutText, stderrText)
	evidence := evidenceFor(projectRoot, runDir)
	candidateMemories, candidateSkills := structuredCandidates(report, stdoutReport, evidence)
	candidateErrors := structuredCandidateErrors(report, stdoutReport)
	candidateErrors = append(candidateErrors, stdoutParse.Errors...)

	proposal := Proposal{
		Version:           1,
		ID:                runID + "-proposal",
		Kind:              "run_reflection",
		Scope:             "personal",
		Operation:         "create",
		Status:            "candidate",
		ArchivistDecision: "defer",
		Safety:            safety,
		Run: RunRef{
			ID:     runID,
			Path:   rel(projectRoot, runDir),
			Status: report.Status,
		},
		Summary:   summary,
		Evidence:  evidence,
		CreatedAt: opts.Now.UTC().Format(time.RFC3339),
		ReviewInstructions: []string{
			"Review in a clean archivist session before applying.",
			"Apply only durable, triggerable memories or reusable skills with evidence.",
			"Reject lessons caused by destructive shortcuts, weakened tests, bypassed validation, or hidden errors.",
		},
	}
	if report.Status == "success" && safety.Decision == "safe" {
		proposal.CandidateMemories = candidateMemories
		proposal.CandidateSkills = candidateSkills
		proposal.CandidateErrors = candidateErrors
		if len(candidateErrors) > 0 {
			proposal.CandidateSource = "structured_invalid"
		} else if len(proposal.CandidateMemories)+len(proposal.CandidateSkills) == 0 {
			proposal.CandidateSource = "fallback"
			proposal.CandidateMemories = []Candidate{{
				Type:     "workflow",
				Title:    "Review successful SIMA run for durable lessons",
				Trigger:  "When learning from a completed SIMA run with captured artifacts.",
				Summary:  "The run completed successfully; inspect the bounded artifact bundle and promote only durable, evidence-backed lessons.",
				Evidence: evidence,
			}}
		} else {
			proposal.CandidateSource = "structured"
		}
	}

	candidateDir := filepath.Join(projectRoot, ".sima", "personal", "memory", "candidates")
	if err := os.MkdirAll(candidateDir, 0o755); err != nil {
		return Result{}, err
	}
	path := filepath.Join(candidateDir, runID+"-proposal.yaml")
	data, err := yaml.Marshal(proposal)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return Result{}, err
	}
	return Result{Path: path, RunID: runID, Decision: proposal.ArchivistDecision, Safety: safety.Decision, Candidates: len(proposal.CandidateMemories) + len(proposal.CandidateSkills), Source: proposal.CandidateSource}, nil
}

func resolveRun(projectRoot, fromRun string) (string, string, error) {
	if fromRun == "last" {
		return latestRun(projectRoot)
	}
	candidates := []string{fromRun}
	if !filepath.IsAbs(fromRun) {
		candidates = append(candidates,
			filepath.Join(projectRoot, fromRun),
			filepath.Join(projectRoot, ".sima", "personal", "runs", fromRun),
		)
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, filepath.Base(candidate), nil
		}
	}
	return "", "", fmt.Errorf("run %q not found", fromRun)
}

func latestRun(projectRoot string) (string, string, error) {
	root := filepath.Join(projectRoot, ".sima", "personal", "runs")
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", "", err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return "", "", fmt.Errorf("no runs found")
	}
	sort.Strings(names)
	name := names[len(names)-1]
	return filepath.Join(root, name), name, nil
}

func readWorkerReport(runDir string) (WorkerReport, error) {
	data, err := os.ReadFile(filepath.Join(runDir, "worker-report.yaml"))
	if err != nil {
		return WorkerReport{}, err
	}
	var report WorkerReport
	if err := yaml.Unmarshal(data, &report); err != nil {
		return WorkerReport{}, err
	}
	return report, nil
}

type parseResult struct {
	Structured bool
	Errors     []string
}

func parseWorkerOutput(stdoutText string) (WorkerReport, parseResult) {
	var report WorkerReport
	text := strings.TrimSpace(stdoutText)
	if text == "" {
		return report, parseResult{}
	}
	if !looksStructured(text) {
		return report, parseResult{}
	}
	if strings.HasPrefix(text, "{") {
		if err := json.Unmarshal([]byte(text), &report); err == nil {
			return report, parseResult{Structured: true}
		}
		return WorkerReport{}, parseResult{Structured: true, Errors: []string{"worker stdout looks like JSON but is not valid JSON"}}
	}
	if err := yaml.Unmarshal([]byte(text), &report); err == nil {
		return report, parseResult{Structured: true}
	}
	return WorkerReport{}, parseResult{Structured: true, Errors: []string{"worker stdout contains proposed_memory/proposed_skills/status but is not valid YAML"}}
}

func structuredCandidates(report, stdoutReport WorkerReport, evidence []Evidence) ([]Candidate, []CandidateSkill) {
	memories := append([]Candidate{}, report.ProposedMemory...)
	memories = append(memories, stdoutReport.ProposedMemory...)
	skills := append([]CandidateSkill{}, report.ProposedSkills...)
	skills = append(skills, stdoutReport.ProposedSkills...)
	for i := range memories {
		if len(memories[i].Evidence) == 0 {
			memories[i].Evidence = evidence
		}
	}
	for i := range skills {
		if len(skills[i].Evidence) == 0 {
			skills[i].Evidence = evidence
		}
	}
	return memories, skills
}

func structuredCandidateErrors(report, stdoutReport WorkerReport) []string {
	var problems []string
	memories := append([]Candidate{}, report.ProposedMemory...)
	memories = append(memories, stdoutReport.ProposedMemory...)
	skills := append([]CandidateSkill{}, report.ProposedSkills...)
	skills = append(skills, stdoutReport.ProposedSkills...)
	for i, c := range memories {
		if strings.TrimSpace(c.Type) == "" || strings.TrimSpace(c.Title) == "" || strings.TrimSpace(c.Trigger) == "" || strings.TrimSpace(c.Summary) == "" {
			problems = append(problems, fmt.Sprintf("proposed_memory[%d] requires type, title, trigger, and summary", i))
		}
	}
	for i, c := range skills {
		if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Trigger) == "" || strings.TrimSpace(c.Summary) == "" {
			problems = append(problems, fmt.Sprintf("proposed_skills[%d] requires name, trigger, and summary", i))
		}
	}
	return problems
}

func looksStructured(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return strings.HasPrefix(line, "{") || line == "proposed_memory:" || line == "proposed_skills:" || strings.HasPrefix(line, "status:")
	}
	return false
}

func readText(path string, limit int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(data) > limit {
		data = data[:limit]
	}
	return string(data)
}

func assessSafety(text string) Safety {
	lower := strings.ToLower(text)
	patterns := map[string]string{
		"weaken test":         "possible test weakening",
		"delete test":         "possible test deletion",
		"removed test":        "possible test removal",
		"skip validation":     "possible validation bypass",
		"bypass validation":   "possible validation bypass",
		"hardcode":            "possible hardcoded output",
		"ignore failure":      "possible hidden failure",
		"ignored failing":     "possible hidden failure",
		"changed requirement": "possible requirement change",
	}
	var flags []string
	for needle, flag := range patterns {
		if strings.Contains(lower, needle) {
			flags = append(flags, flag)
		}
	}
	sort.Strings(flags)
	if len(flags) > 0 {
		return Safety{Decision: "suspicious", Flags: flags}
	}
	return Safety{Decision: "safe"}
}

func summarize(report WorkerReport, stdoutText, stderrText string) string {
	if report.Status == "failed" || report.ExitCode != 0 {
		line := firstNonEmpty(stderrText)
		if line == "" {
			line = firstNonEmpty(stdoutText)
		}
		if line == "" {
			line = "Run failed; inspect logs before proposing lessons."
		}
		return line
	}
	line := firstNonEmpty(stdoutText)
	if line == "" {
		line = "Run completed; inspect artifacts before promoting durable lessons."
	}
	return line
}

func firstNonEmpty(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) > 240 {
				line = line[:240]
			}
			return line
		}
	}
	return ""
}

func evidenceFor(projectRoot, runDir string) []Evidence {
	items := []Evidence{}
	for _, item := range []struct {
		kind string
		name string
		note string
	}{
		{"task", "task.md", "original task"},
		{"brief", "brief.md", "pre-task briefing"},
		{"command", "command.txt", "backend command"},
		{"stdout", "stdout.log", "worker stdout"},
		{"stderr", "stderr.log", "worker stderr"},
		{"worker_report", "worker-report.yaml", "structured worker report"},
	} {
		path := filepath.Join(runDir, item.name)
		if _, err := os.Stat(path); err == nil {
			items = append(items, Evidence{Kind: item.kind, Path: rel(projectRoot, path), Note: item.note})
		}
	}
	return items
}

func rel(projectRoot, path string) string {
	rel, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
