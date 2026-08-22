package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/antowkos/sima/internal/brief"
	"github.com/antowkos/sima/internal/config"
)

type Options struct {
	BackendName string
	Task        string
	Now         time.Time
}

type Result struct {
	RunID    string
	RunDir   string
	ExitCode int
}

type WorkerReport struct {
	RunID       string   `yaml:"run_id"`
	Backend     string   `yaml:"backend"`
	Status      string   `yaml:"status"`
	Task        string   `yaml:"task"`
	BriefPath   string   `yaml:"brief_path"`
	Command     []string `yaml:"command"`
	StartedAt   string   `yaml:"started_at"`
	CompletedAt string   `yaml:"completed_at"`
	ExitCode    int      `yaml:"exit_code"`
	StdoutPath  string   `yaml:"stdout_path"`
	StderrPath  string   `yaml:"stderr_path"`
}

func Run(projectRoot string, opts Options) (Result, error) {
	if strings.TrimSpace(opts.BackendName) == "" {
		return Result{}, fmt.Errorf("backend is required")
	}
	if strings.TrimSpace(opts.Task) == "" {
		return Result{}, fmt.Errorf("task is required")
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return Result{}, err
	}
	profile, ok := cfg.Backends[opts.BackendName]
	if !ok {
		return Result{}, fmt.Errorf("backend %q not found", opts.BackendName)
	}

	briefResult, err := brief.Generate(projectRoot, brief.Options{Task: opts.Task, Now: opts.Now})
	if err != nil {
		return Result{}, err
	}

	runID := opts.Now.UTC().Format("20060102-150405") + "-" + sanitize(opts.BackendName)
	runDir := filepath.Join(projectRoot, ".sima", "personal", "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return Result{}, err
	}

	if err := os.WriteFile(filepath.Join(runDir, "task.md"), []byte(opts.Task+"\n"), 0o644); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(filepath.Join(runDir, "brief.md"), []byte(briefResult.Content), 0o644); err != nil {
		return Result{}, err
	}

	prompt := buildPrompt(opts.Task, filepath.Join(runDir, "brief.md"))
	argv := buildArgs(profile, prompt)
	commandLine := append([]string{profile.Executable}, argv...)
	if err := os.WriteFile(filepath.Join(runDir, "command.txt"), []byte(strings.Join(commandLine, " ")+"\n"), 0o644); err != nil {
		return Result{}, err
	}

	cmd := exec.Command(expandHome(profile.Executable), argv...)
	cmd.Dir = projectRoot
	if profile.WorkingDir != "" {
		cmd.Dir = expandHome(profile.WorkingDir)
	}
	cmd.Env = mergeEnv(os.Environ(), profile.Env)

	stdoutBytes, stderrBytes, exitCode := runCommand(cmd)
	stdoutPath := filepath.Join(runDir, "stdout.log")
	stderrPath := filepath.Join(runDir, "stderr.log")
	if err := os.WriteFile(stdoutPath, stdoutBytes, 0o644); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(stderrPath, stderrBytes, 0o644); err != nil {
		return Result{}, err
	}

	status := "success"
	if exitCode != 0 {
		status = "failed"
	}
	report := WorkerReport{
		RunID:       runID,
		Backend:     opts.BackendName,
		Status:      status,
		Task:        opts.Task,
		BriefPath:   rel(projectRoot, filepath.Join(runDir, "brief.md")),
		Command:     commandLine,
		StartedAt:   opts.Now.UTC().Format(time.RFC3339),
		CompletedAt: time.Now().UTC().Format(time.RFC3339),
		ExitCode:    exitCode,
		StdoutPath:  rel(projectRoot, stdoutPath),
		StderrPath:  rel(projectRoot, stderrPath),
	}
	reportBytes, err := yaml.Marshal(report)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(filepath.Join(runDir, "worker-report.yaml"), reportBytes, 0o644); err != nil {
		return Result{}, err
	}

	return Result{RunID: runID, RunDir: runDir, ExitCode: exitCode}, nil
}

func buildPrompt(task, briefPath string) string {
	return fmt.Sprintf("Read the SIMA brief at %s, then perform this bounded task and return a concise structured report: %s", briefPath, task)
}

func buildArgs(profile config.BackendProfile, prompt string) []string {
	switch profile.Kind {
	case "claude-code":
		return []string{"-p", prompt}
	case "codex":
		return []string{"exec", prompt}
	default:
		return []string{prompt}
	}
}

func runCommand(cmd *exec.Cmd) ([]byte, []byte, int) {
	var exitCode int
	stdoutBytes, err := cmd.Output()
	if err == nil {
		return stdoutBytes, nil, 0
	}
	stderrBytes := []byte(err.Error() + "\n")
	if ee, ok := err.(*exec.ExitError); ok {
		stderrBytes = append(ee.Stderr, stderrBytes...)
		exitCode = ee.ExitCode()
	} else {
		exitCode = 1
	}
	return stdoutBytes, stderrBytes, exitCode
}

func mergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	merged := append([]string{}, base...)
	for k, v := range extra {
		merged = append(merged, k+"="+v)
	}
	return merged
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func sanitize(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
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
