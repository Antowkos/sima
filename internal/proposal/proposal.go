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
	RunID            string           `yaml:"run_id" json:"run_id,omitempty"`
	Backend          string           `yaml:"backend" json:"backend,omitempty"`
	Status           string           `yaml:"status" json:"status,omitempty"`
	Task             string           `yaml:"task" json:"task,omitempty"`
	BriefPath        string           `yaml:"brief_path" json:"brief_path,omitempty"`
	ExitCode         int              `yaml:"exit_code" json:"exit_code,omitempty"`
	StdoutPath       string           `yaml:"stdout_path" json:"stdout_path,omitempty"`
	StderrPath       string           `yaml:"stderr_path" json:"stderr_path,omitempty"`
	ProposedMemory   []Candidate      `yaml:"proposed_memory,omitempty" json:"proposed_memory,omitempty"`
	ProposedSkills   []CandidateSkill `yaml:"proposed_skills,omitempty" json:"proposed_skills,omitempty"`
	StructuredOutput *WorkerReport    `yaml:"-" json:"structured_output,omitempty"`
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
	Learning           Learning         `yaml:"learning,omitempty"`
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

type Learning struct {
	Destination string          `yaml:"destination,omitempty" json:"destination,omitempty"`
	Operation   string          `yaml:"operation,omitempty" json:"operation,omitempty"`
	Target      LearningTarget  `yaml:"target,omitempty" json:"target,omitempty"`
	Quality     LearningQuality `yaml:"quality,omitempty" json:"quality,omitempty"`
	Notes       []string        `yaml:"notes,omitempty" json:"notes,omitempty"`
}

type LearningTarget struct {
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`
	Path string `yaml:"path,omitempty" json:"path,omitempty"`
	ID   string `yaml:"id,omitempty" json:"id,omitempty"`
}

type LearningQuality struct {
	Durable        bool `yaml:"durable" json:"durable"`
	Triggerable    bool `yaml:"triggerable" json:"triggerable"`
	EvidenceBacked bool `yaml:"evidence_backed" json:"evidence_backed"`
	NonTransient   bool `yaml:"non_transient" json:"non_transient"`
	Reusable       bool `yaml:"reusable" json:"reusable"`
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
	proposal.Learning = classifyLearning(projectRoot, proposal)

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
			if report.StructuredOutput != nil {
				return *report.StructuredOutput, parseResult{Structured: true}
			}
			return report, parseResult{Structured: true}
		}
		return WorkerReport{}, parseResult{Structured: true, Errors: []string{"worker stdout looks like JSON but is not valid JSON"}}
	}
	return WorkerReport{}, parseResult{Structured: true, Errors: []string{"worker stdout must be JSON and start with '{'"}}
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

func classifyLearning(projectRoot string, p Proposal) Learning {
	learning := Learning{
		Destination: "session_only",
		Operation:   p.Operation,
		Quality: LearningQuality{
			Durable:        true,
			Triggerable:    true,
			EvidenceBacked: len(p.Evidence) > 0,
			NonTransient:   true,
			Reusable:       true,
		},
	}
	if p.CandidateSource == "structured_invalid" || len(p.CandidateErrors) > 0 {
		learning.Destination = "reject"
		learning.Quality.Durable = false
		learning.Quality.Triggerable = false
		learning.Quality.Reusable = false
		learning.Notes = append(learning.Notes, "structured output is malformed or incomplete")
		return learning
	}
	if p.CandidateSource == "fallback" {
		learning.Destination = "session_only"
		learning.Quality.Durable = false
		learning.Quality.Reusable = false
		learning.Notes = append(learning.Notes, "fallback candidates require human/librarian review before promotion")
		return learning
	}
	memoryCount := len(p.CandidateMemories)
	skillCount := len(p.CandidateSkills)
	if memoryCount == 0 && skillCount == 0 {
		learning.Destination = "session_only"
		learning.Quality.Durable = false
		learning.Quality.Triggerable = false
		learning.Quality.Reusable = false
		learning.Notes = append(learning.Notes, "no candidate memory or skill was proposed")
		return learning
	}
	switch {
	case memoryCount > 0 && skillCount > 0:
		learning.Destination = "mixed"
	case skillCount > 0:
		learning.Destination = "skill"
	case memoryCount > 0:
		learning.Destination = "memory"
	}
	for i, c := range p.CandidateMemories {
		text := c.Title + "\n" + c.Trigger + "\n" + c.Summary
		if !looksTriggerable(c.Trigger) {
			learning.Quality.Triggerable = false
			learning.Notes = append(learning.Notes, fmt.Sprintf("candidate_memories[%d] trigger does not describe recall timing", i))
		}
		if containsTransientLesson(text) {
			learning.Quality.Durable = false
			learning.Quality.NonTransient = false
			learning.Notes = append(learning.Notes, fmt.Sprintf("candidate_memories[%d] looks transient", i))
		}
		if len(c.Evidence) == 0 {
			learning.Quality.EvidenceBacked = false
			learning.Notes = append(learning.Notes, fmt.Sprintf("candidate_memories[%d] missing evidence", i))
		}
	}
	for i, c := range p.CandidateSkills {
		text := c.Name + "\n" + c.Trigger + "\n" + c.Summary
		if !looksTriggerable(c.Trigger) {
			learning.Quality.Triggerable = false
			learning.Notes = append(learning.Notes, fmt.Sprintf("candidate_skills[%d] trigger does not describe usage timing", i))
		}
		if containsTransientLesson(text) {
			learning.Quality.Durable = false
			learning.Quality.NonTransient = false
			learning.Notes = append(learning.Notes, fmt.Sprintf("candidate_skills[%d] looks transient", i))
		}
		if len(c.Evidence) == 0 {
			learning.Quality.EvidenceBacked = false
			learning.Notes = append(learning.Notes, fmt.Sprintf("candidate_skills[%d] missing evidence", i))
		}
	}
	if skillCount == 0 {
		learning.Quality.Reusable = true
	}
	learning = inferLifecycleTarget(projectRoot, learning, p)
	return learning
}

type activeMemoryRef struct {
	ID          string
	Path        string
	TitleSlug   string
	TriggerSlug string
}

type activeSkillRef struct {
	ID          string
	Path        string
	NameSlug    string
	TriggerSlug string
}

func inferLifecycleTarget(projectRoot string, learning Learning, p Proposal) Learning {
	if learning.Operation != "create" || !learningQualityPasses(learning.Quality) {
		return learning
	}
	if len(p.CandidateMemories) == 1 && len(p.CandidateSkills) == 0 {
		candidate := p.CandidateMemories[0]
		if match, ok := matchActiveMemory(projectRoot, candidate); ok {
			learning.Operation = "update"
			learning.Target = LearningTarget{Kind: "memory", Path: match.Path, ID: match.ID}
			learning.Notes = append(learning.Notes, "similar active memory found; classify as update")
		}
	}
	if len(p.CandidateSkills) == 1 && len(p.CandidateMemories) == 0 {
		candidate := p.CandidateSkills[0]
		if match, ok := matchActiveSkill(projectRoot, candidate); ok {
			learning.Operation = "update"
			learning.Target = LearningTarget{Kind: "skill", Path: match.Path, ID: match.ID}
			learning.Notes = append(learning.Notes, "similar active skill found; classify as update")
		}
	}
	return learning
}

func matchActiveMemory(projectRoot string, c Candidate) (activeMemoryRef, bool) {
	candidateTitle := slug(c.Title)
	candidateTrigger := slug(c.Trigger)
	for _, active := range readActiveMemory(projectRoot) {
		if candidateTitle != "" && candidateTitle == active.TitleSlug {
			return active, true
		}
		if candidateTrigger != "" && candidateTrigger == active.TriggerSlug {
			return active, true
		}
	}
	return activeMemoryRef{}, false
}

func matchActiveSkill(projectRoot string, c CandidateSkill) (activeSkillRef, bool) {
	candidateName := slug(c.Name)
	candidateTrigger := slug(c.Trigger)
	for _, active := range readActiveSkills(projectRoot) {
		if candidateName != "" && candidateName == active.NameSlug {
			return active, true
		}
		if candidateTrigger != "" && candidateTrigger == active.TriggerSlug {
			return active, true
		}
	}
	return activeSkillRef{}, false
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
			ID      string `yaml:"id"`
			Title   string `yaml:"title"`
			Trigger string `yaml:"trigger"`
			Status  string `yaml:"status"`
		}
		if err := yaml.Unmarshal(data, &card); err != nil || !isActiveStatus(card.Status) {
			continue
		}
		refs = append(refs, activeMemoryRef{ID: card.ID, Path: rel(projectRoot, path), TitleSlug: slug(card.Title), TriggerSlug: slug(card.Trigger)})
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
		refs = append(refs, activeSkillRef{ID: name, Path: rel(projectRoot, path), NameSlug: slug(name), TriggerSlug: slug(extractSkillTrigger(string(data)))})
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

func isActiveStatus(status string) bool {
	return status == "" || status == "active"
}

func isYAML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}

func learningQualityPasses(q LearningQuality) bool {
	return q.Durable && q.Triggerable && q.EvidenceBacked && q.NonTransient && q.Reusable
}

func looksTriggerable(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(lower, "when ") || strings.Contains(lower, "when_") || strings.Contains(lower, "if ") || strings.Contains(lower, "before ") || strings.Contains(lower, "after ") || strings.Contains(lower, "while ") || strings.Contains(lower, "during ")
}

func containsTransientLesson(text string) bool {
	lower := strings.ToLower(text)
	transient := []string{"commit ", "committed ", "pushed ", "pr #", "pull request", "issue #", "today ", "yesterday ", "just now", "this run", "this task", "applied proposal", "smoke test completed"}
	for _, marker := range transient {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
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
