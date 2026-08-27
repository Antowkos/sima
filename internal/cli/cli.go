package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antowkos/sima/internal/apply"
	"github.com/antowkos/sima/internal/archivist"
	"github.com/antowkos/sima/internal/backend"
	"github.com/antowkos/sima/internal/brief"
	"github.com/antowkos/sima/internal/candidates"
	"github.com/antowkos/sima/internal/catalog"
	"github.com/antowkos/sima/internal/config"
	"github.com/antowkos/sima/internal/lint"
	"github.com/antowkos/sima/internal/proposal"
	"github.com/antowkos/sima/internal/review"
	"github.com/antowkos/sima/internal/runner"
	"github.com/antowkos/sima/internal/simafs"
)

var Version = "0.1.0-dev"

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		printHelp(stdout)
		return 0
	}

	switch args[1] {
	case "help", "--help", "-h":
		printHelp(stdout)
		return 0
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "sima %s\n", Version)
		return 0
	case "init":
		return runInit(args[2:], stdout, stderr)
	case "install":
		return runInstall(args[2:], stdout, stderr)
	case "setup":
		return runSetup(args[2:], stdout, stderr)
	case "doctor":
		return runDoctor(args[2:], stdout, stderr)
	case "lint":
		return runLint(args[2:], stdout, stderr)
	case "backend":
		return runBackend(args[2:], stdout, stderr)
	case "brief":
		return runBrief(args[2:], stdout, stderr)
	case "run":
		return runRun(args[2:], stdout, stderr)
	case "learn":
		return runLearn(args[2:], stdout, stderr)
	case "remember":
		return runRemember(args[2:], stdout, stderr)
	case "propose":
		return runPropose(args[2:], stdout, stderr)
	case "review":
		return runReview(args[2:], stdout, stderr)
	case "apply":
		return runApply(args[2:], stdout, stderr)
	case "archivist":
		return runArchivist(args[2:], stdout, stderr)
	case "candidates":
		return runCandidates(args[2:], stdout, stderr)
	case "memory":
		return runCatalogCommand("memory", args[2:], stdout, stderr)
	case "skill":
		return runCatalogCommand("skill", args[2:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[1])
		printHelp(stderr)
		return 2
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `SIMA — Self Improvement Memory Agent

Usage:
  sima init [path]
  sima install [--client <claude|codex|all>] [--path <path>]
  sima setup [path] [--path <path>] [--backend <auto|claude|codex|none>] [--executable <path>] [--claude-executable <path>] [--codex-executable <path>] [--claude-config-dir <path>] [--env KEY=VALUE]
  sima doctor [path]
  sima lint [path]
  sima brief <task> [--path <path>]
  sima run --backend <name> --task <task> [--path <path>] [--no-propose]
  sima learn --backend <name> --task <task> [--archivist-backend <name>] [--auto-apply|--no-auto-apply] [--auto-cleanup-deferred|--no-auto-cleanup-deferred] [--json] [--path <path>]
  sima remember <knowledge> [--source <user|review|agent>] [--type <memory-type>] [--title <title>] [--trigger <trigger>] [--backend <name>] [--path <path>]
  sima propose --from-run <run-id|last|path> [--path <path>]
  sima review [--path <path>] [--all]
  sima apply <proposal-id|path> [--path <path>]
  sima archivist --proposal <proposal-id|path> [--backend <backend>] [--path <path>]
  sima candidates list [--status <candidate|deferred|applied|rejected|all>] [--path <path>]
  sima candidates apply-ready [--apply] [--path <path>]
  sima candidates show <id|path> [--path <path>]
  sima candidates cleanup [--path <path>]
  sima memory list [--status <active|deprecated|superseded|archived|all>] [--path <path>]
  sima skill list [--status <active|deprecated|superseded|archived|all>] [--path <path>]
  sima backend list [path]
  sima backend add <name> --kind <claude-code|codex> --executable <path> [options]
  sima backend doctor <name> [path]
  sima version

Current v0 slice:
  init     Create project-local .sima scaffold
  install  Upsert managed Claude Code/Codex project instruction blocks
  setup    Initialize project state, install instructions, optionally add backend, and run preflight
  doctor   Check SIMA project state and local runtime prerequisites
  lint     Check SIMA knowledge lifecycle metadata, candidates, and target paths
  brief    Create a compact task briefing from SIMA memory, skills, and SDD artifacts
  run      Run a bounded task through a named backend, capture artifacts, and auto-propose candidates
  learn    Run the full gated self-improvement loop: run, propose, archivist, apply-ready, apply
  remember Capture explicit user/review/agent knowledge as a SIMA candidate, optionally archivist/apply it
  propose  Create a candidate proposal from a captured run bundle
  review   Validate and summarize pending candidate proposals
  candidates List, inspect, and clean candidate proposals
  apply    Promote an approved safe personal proposal into active memory/skills
  archivist Decide apply/reject/defer for a candidate proposal with deterministic gates
  backend  Manage named Claude Code/Codex backend profiles`)
}

func runInit(args []string, stdout, stderr io.Writer) int {
	target := "."
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: sima init [path]")
		return 2
	}
	if len(args) == 1 {
		target = args[0]
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}

	created, err := simafs.Init(abs)
	if err != nil {
		fmt.Fprintf(stderr, "init failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Initialized SIMA state at %s\n", filepath.Join(abs, ".sima"))
	if len(created) > 0 {
		fmt.Fprintln(stdout, "Created:")
		for _, p := range created {
			fmt.Fprintf(stdout, "  - %s\n", p)
		}
	}
	return 0
}

func runInstall(args []string, stdout, stderr io.Writer) int {
	root := "."
	client := "all"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			root = args[i+1]
			i++
		case "--client":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--client requires a value")
				return 2
			}
			client = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", args[i])
			return 2
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	clients := []string{}
	switch strings.ToLower(client) {
	case "all":
		clients = nil
	case "claude", "codex":
		clients = []string{client}
	default:
		fmt.Fprintf(stderr, "unknown client: %s\n", client)
		return 2
	}
	result, err := simafs.InstallInstructions(abs, simafs.InstallOptions{Clients: clients})
	if err != nil {
		fmt.Fprintf(stderr, "install failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Installed SIMA managed instructions in %s\n", abs)
	for _, path := range result.Written {
		fmt.Fprintf(stdout, "  - %s\n", path)
	}
	return 0
}

func runSetup(args []string, stdout, stderr io.Writer) int {
	root := "."
	rootSet := false
	backendMode := "auto"
	backendExecutable := ""
	claudeExecutable := ""
	codexExecutable := ""
	profileEnv := map[string]string{}
	profileConfig := ""
	profileEnvFile := ""
	profileWorkingDir := ""
	profilePermissionMode := ""
	profileMetadata := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			if rootSet {
				fmt.Fprintln(stderr, "setup path specified more than once")
				return 2
			}
			i++
			root = args[i]
			rootSet = true
		case "--backend":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--backend requires a value")
				return 2
			}
			i++
			backendMode = strings.ToLower(args[i])
		case "--executable", "--backend-executable":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "%s requires a value\n", args[i])
				return 2
			}
			i++
			backendExecutable = args[i]
		case "--claude-executable":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--claude-executable requires a value")
				return 2
			}
			i++
			claudeExecutable = args[i]
		case "--codex-executable":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--codex-executable requires a value")
				return 2
			}
			i++
			codexExecutable = args[i]
		case "--claude-config-dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--claude-config-dir requires a value")
				return 2
			}
			i++
			profileEnv["CLAUDE_CONFIG_DIR"] = expandSetupHome(args[i])
		case "--config":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--config requires a value")
				return 2
			}
			i++
			profileConfig = args[i]
		case "--env-file":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--env-file requires a value")
				return 2
			}
			i++
			profileEnvFile = args[i]
		case "--working-dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--working-dir requires a value")
				return 2
			}
			i++
			profileWorkingDir = args[i]
		case "--permission-mode":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--permission-mode requires a value")
				return 2
			}
			i++
			profilePermissionMode = args[i]
		case "--env":
			if i+1 >= len(args) || !strings.Contains(args[i+1], "=") {
				fmt.Fprintln(stderr, "--env requires KEY=VALUE")
				return 2
			}
			i++
			parts := strings.SplitN(args[i], "=", 2)
			profileEnv[parts[0]] = parts[1]
		case "--metadata":
			if i+1 >= len(args) || !strings.Contains(args[i+1], "=") {
				fmt.Fprintln(stderr, "--metadata requires KEY=VALUE")
				return 2
			}
			i++
			parts := strings.SplitN(args[i], "=", 2)
			profileMetadata[parts[0]] = parts[1]
		case "--help", "-h":
			fmt.Fprintln(stdout, "usage: sima setup [path] [--path <path>] [--backend <auto|claude|codex|none>] [--executable <path>] [--claude-executable <path>] [--codex-executable <path>] [--claude-config-dir <path>] [--env KEY=VALUE]")
			return 0
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown option: %s\n", args[i])
				return 2
			}
			if rootSet {
				fmt.Fprintf(stderr, "unexpected extra argument: %s\n", args[i])
				return 2
			}
			root = args[i]
			rootSet = true
		}
	}
	if backendExecutable != "" && backendMode == "auto" {
		fmt.Fprintln(stderr, "--executable requires --backend claude or --backend codex; use --claude-executable/--codex-executable with --backend auto")
		return 2
	}
	switch backendMode {
	case "auto", "claude", "codex", "none":
	default:
		fmt.Fprintf(stderr, "unknown backend mode: %s\n", backendMode)
		return 2
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Setting up SIMA project in %s\n", abs)
	if _, err := simafs.Init(abs); err != nil {
		fmt.Fprintf(stderr, "init failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "Initialized project state")

	result, err := simafs.InstallInstructions(abs, simafs.InstallOptions{})
	if err != nil {
		fmt.Fprintf(stderr, "install instructions failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "Installed managed instructions")
	for _, path := range result.Written {
		fmt.Fprintf(stdout, "  - %s\n", path)
	}

	selectedBackend, backendName, backendProfile, err := selectSetupBackend(backendMode, setupBackendExecutables{
		Backend: backendExecutable,
		Claude:  claudeExecutable,
		Codex:   codexExecutable,
	})
	if err != nil {
		fmt.Fprintf(stderr, "backend setup failed: %v\n", err)
		return 1
	}
	backendProfile = applySetupBackendProfileOptions(backendProfile, setupBackendProfileOptions{
		ConfigPath:     profileConfig,
		EnvFile:        profileEnvFile,
		WorkingDir:     profileWorkingDir,
		PermissionMode: profilePermissionMode,
		Env:            profileEnv,
		Metadata:       profileMetadata,
	})
	if selectedBackend == "none" {
		fmt.Fprintln(stdout, "Skipped backend setup. Add one later with: sima backend add ...")
		fmt.Fprintln(stdout, "Running lint preflight...")
		if runLint([]string{abs}, stdout, stderr) != 0 {
			return 1
		}
		fmt.Fprintln(stdout, "Skipped sima doctor because no backend is configured yet. Run it after sima backend add.")
	} else {
		if err := config.AddBackend(abs, backendName, backendProfile, true); err != nil {
			fmt.Fprintf(stderr, "add backend: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Added backend: %s\n", backendName)
		if selectedBackend == "codex" {
			fmt.Fprintln(stdout, "Before the first Codex learn run, check auth with: codex doctor")
		}
		fmt.Fprintln(stdout, "Running preflight...")
		if runDoctor([]string{abs}, stdout, stderr) != 0 {
			return 1
		}
	}

	fmt.Fprintf(stdout, "Next: sima brief \"small real task\" --path %s\n", abs)
	return 0
}

type setupBackendExecutables struct {
	Backend string
	Claude  string
	Codex   string
}

type setupBackendProfileOptions struct {
	ConfigPath     string
	EnvFile        string
	WorkingDir     string
	PermissionMode string
	Env            map[string]string
	Metadata       map[string]string
}

func applySetupBackendProfileOptions(profile config.BackendProfile, opts setupBackendProfileOptions) config.BackendProfile {
	if opts.ConfigPath != "" {
		profile.ConfigPath = opts.ConfigPath
	}
	if opts.EnvFile != "" {
		profile.EnvFile = opts.EnvFile
	}
	if opts.WorkingDir != "" {
		profile.WorkingDir = opts.WorkingDir
	}
	if opts.PermissionMode != "" {
		profile.PermissionMode = opts.PermissionMode
	}
	if len(opts.Env) > 0 {
		profile.Env = opts.Env
	}
	if len(opts.Metadata) > 0 {
		profile.Metadata = opts.Metadata
	}
	return profile
}

func selectSetupBackend(mode string, executables setupBackendExecutables) (selected string, name string, profile config.BackendProfile, err error) {
	if mode == "auto" {
		if executable, lookErr := resolveSetupExecutable("claude", executables.Claude); lookErr == nil {
			return "claude", "claude-main", config.BackendProfile{Kind: "claude-code", Executable: executable}, nil
		} else if executables.Claude != "" {
			return "", "", config.BackendProfile{}, fmt.Errorf("--claude-executable %q was not found", executables.Claude)
		}
		if executable, lookErr := resolveSetupExecutable("codex", executables.Codex); lookErr == nil {
			return "codex", "codex-main", config.BackendProfile{Kind: "codex", Executable: executable, PermissionMode: "workspace-write"}, nil
		} else if executables.Codex != "" {
			return "", "", config.BackendProfile{}, fmt.Errorf("--codex-executable %q was not found", executables.Codex)
		}
		return "none", "", config.BackendProfile{}, nil
	}
	if mode == "none" {
		return "none", "", config.BackendProfile{}, nil
	}
	explicitExecutable := executables.Backend
	if explicitExecutable == "" && mode == "claude" {
		explicitExecutable = executables.Claude
	}
	if explicitExecutable == "" && mode == "codex" {
		explicitExecutable = executables.Codex
	}
	executable, lookErr := resolveSetupExecutable(mode, explicitExecutable)
	if lookErr != nil {
		if explicitExecutable != "" {
			return "", "", config.BackendProfile{}, fmt.Errorf("--backend %s requested but executable %q was not found", mode, explicitExecutable)
		}
		return "", "", config.BackendProfile{}, fmt.Errorf("--backend %s requested but %s was not found in PATH", mode, mode)
	}
	if mode == "claude" {
		return "claude", "claude-main", config.BackendProfile{Kind: "claude-code", Executable: executable}, nil
	}
	return "codex", "codex-main", config.BackendProfile{Kind: "codex", Executable: executable, PermissionMode: "workspace-write"}, nil
}

func resolveSetupExecutable(defaultName, explicit string) (string, error) {
	if explicit != "" {
		return exec.LookPath(explicit)
	}
	return exec.LookPath(defaultName)
}

func expandSetupHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	target := "."
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: sima doctor [path]")
		return 2
	}
	if len(args) == 1 {
		target = args[0]
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}

	report := simafs.Doctor(abs)
	fmt.Fprintf(stdout, "SIMA doctor for %s\n", abs)
	failed := false
	for _, check := range report.Checks {
		if !printDoctorCheck(stdout, check.OK, "project: "+check.Name, check.Detail) {
			failed = true
		}
	}

	cfg, err := config.Load(abs)
	if err != nil {
		printDoctorCheck(stdout, false, "config: load", err.Error())
		failed = true
	} else {
		printDoctorCheck(stdout, true, "config: load", "version loaded")
		learnDetail := fmt.Sprintf("auto_apply=%t auto_cleanup_deferred=%t", cfg.Learn.AutoApply, cfg.Learn.AutoCleanupDeferred)
		if !printDoctorCheck(stdout, cfg.Learn.AutoApply, "config: learn auto_apply", learnDetail) {
			failed = true
		}
		if !printDoctorCheck(stdout, cfg.Learn.AutoCleanupDeferred, "config: learn auto_cleanup_deferred", learnDetail) {
			failed = true
		}

		backendNames := make([]string, 0, len(cfg.Backends))
		for name := range cfg.Backends {
			backendNames = append(backendNames, name)
		}
		sort.Strings(backendNames)
		if len(backendNames) == 0 {
			printDoctorCheck(stdout, false, "backends: configured", "no backends configured; run sima backend add")
			failed = true
		} else {
			printDoctorCheck(stdout, true, "backends: configured", fmt.Sprintf("%d backend(s): %s", len(backendNames), strings.Join(backendNames, ", ")))
		}
		for _, name := range backendNames {
			result := backend.Doctor(name, cfg.Backends[name])
			if !printDoctorCheck(stdout, result.OK, "backend: "+name, result.Detail) {
				failed = true
			}
		}
	}

	lintResult, err := lint.Check(abs)
	if err != nil {
		printDoctorCheck(stdout, false, "lint: check", err.Error())
		failed = true
	} else {
		lintOK := lintResult.ErrorCount() == 0
		if !printDoctorCheck(stdout, lintOK, "lint: errors", fmt.Sprintf("%d errors, %d warnings", lintResult.ErrorCount(), lintResult.WarningCount())) {
			failed = true
		}
		if lintResult.WarningCount() > 0 {
			fmt.Fprintf(stdout, "[warn] lint: warnings: %d warnings; inspect with sima lint %s\n", lintResult.WarningCount(), abs)
		}
	}

	candidateItems, err := candidates.List(abs, candidates.ListOptions{Status: "candidate"})
	if err != nil {
		printDoctorCheck(stdout, false, "candidates: queue", err.Error())
		failed = true
	} else if len(candidateItems) == 0 {
		printDoctorCheck(stdout, true, "candidates: queue", "0 pending candidates")
	} else {
		fmt.Fprintf(stdout, "[warn] candidates: queue: %d pending candidate(s); inspect with sima candidates list --path %s\n", len(candidateItems), abs)
	}

	if failed {
		fmt.Fprintln(stderr, "SIMA doctor found problems")
		return 1
	}
	return 0
}

func printDoctorCheck(w io.Writer, ok bool, name, detail string) bool {
	mark := "ok"
	if !ok {
		mark = "fail"
	}
	fmt.Fprintf(w, "[%s] %s", mark, name)
	if detail != "" {
		fmt.Fprintf(w, ": %s", detail)
	}
	fmt.Fprintln(w)
	return ok
}

func runLint(args []string, stdout, stderr io.Writer) int {
	root := "."
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: sima lint [path]")
		return 2
	}
	if len(args) == 1 {
		root = args[0]
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	result, err := lint.Check(abs)
	if err != nil {
		fmt.Fprintf(stderr, "lint failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "SIMA lint for %s\n", abs)
	for _, issue := range result.Issues {
		fmt.Fprintf(stdout, "[%s] %s: %s\n", issue.Severity, issue.Path, issue.Message)
	}
	fmt.Fprintf(stdout, "Summary: %d errors, %d warnings\n", result.ErrorCount(), result.WarningCount())
	if result.ErrorCount() > 0 {
		return 1
	}
	return 0
}

func runBrief(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: sima brief <task> [--path <path>]")
		return 2
	}
	root := "."
	var taskParts []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		default:
			taskParts = append(taskParts, arg)
		}
	}
	if len(taskParts) == 0 {
		fmt.Fprintln(stderr, "task is required")
		return 2
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	result, err := brief.Generate(abs, brief.Options{Task: strings.Join(taskParts, " ")})
	if err != nil {
		fmt.Fprintf(stderr, "brief failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Brief written: %s\n", result.Path)
	return 0
}

func runRun(args []string, stdout, stderr io.Writer) int {
	root := "."
	backendName := ""
	task := ""
	autoPropose := true
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--backend":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--backend requires a value")
				return 2
			}
			i++
			backendName = args[i]
		case "--task":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--task requires a value")
				return 2
			}
			i++
			task = args[i]
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		case "--no-propose":
			autoPropose = false
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", arg)
			return 2
		}
	}
	if backendName == "" || task == "" {
		fmt.Fprintln(stderr, "usage: sima run --backend <name> --task <task> [--path <path>] [--no-propose]")
		return 2
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	result, err := runner.Run(abs, runner.Options{BackendName: backendName, Task: task})
	if err != nil {
		fmt.Fprintf(stderr, "run failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Run %s complete: %s\n", result.RunID, result.RunDir)
	fmt.Fprintf(stdout, "Exit code: %d\n", result.ExitCode)
	if result.ExitCode != 0 {
		return 1
	}
	if autoPropose {
		proposalResult, err := proposal.Generate(abs, proposal.Options{FromRun: result.RunID})
		if err != nil {
			fmt.Fprintf(stderr, "auto-propose failed: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Proposal written: %s\n", proposalResult.Path)
		fmt.Fprintf(stdout, "Candidates: %d\n", proposalResult.Candidates)
		if proposalResult.Source != "" {
			fmt.Fprintf(stdout, "Candidate source: %s\n", proposalResult.Source)
		}
		fmt.Fprintf(stdout, "Next: sima review --path %s\n", abs)
		fmt.Fprintf(stdout, "Next: sima archivist --proposal %s --path %s\n", filepath.Base(strings.TrimSuffix(proposalResult.Path, ".yaml")), abs)
	}
	return 0
}

func runLearn(args []string, stdout, stderr io.Writer) int {
	root := "."
	backendName := ""
	archivistBackendName := ""
	task := ""
	jsonOutput := false
	var autoApplyOverride *bool
	var autoCleanupDeferredOverride *bool
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--backend":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--backend requires a value")
				return 2
			}
			i++
			backendName = args[i]
		case "--archivist-backend", "--reviewer":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "%s requires a value\n", arg)
				return 2
			}
			i++
			archivistBackendName = args[i]
		case "--task":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--task requires a value")
				return 2
			}
			i++
			task = args[i]
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		case "--no-auto-apply":
			autoApplyOverride = boolPtr(false)
		case "--auto-apply":
			autoApplyOverride = boolPtr(true)
		case "--auto-cleanup-deferred":
			autoCleanupDeferredOverride = boolPtr(true)
		case "--no-auto-cleanup-deferred":
			autoCleanupDeferredOverride = boolPtr(false)
		case "--json":
			jsonOutput = true
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", arg)
			return 2
		}
	}
	if backendName == "" || task == "" {
		fmt.Fprintln(stderr, "usage: sima learn --backend <name> --task <task> [--archivist-backend <name>] [--auto-apply|--no-auto-apply] [--auto-cleanup-deferred|--no-auto-cleanup-deferred] [--json] [--path <path>]")
		return 2
	}
	humanOut := stdout
	if jsonOutput {
		humanOut = io.Discard
	}
	summary := learnSummary{Status: "running", Outcome: "started", Backend: backendName, ArchivistBackend: archivistBackendName, Task: task}
	if archivistBackendName == "" {
		archivistBackendName = backendName
	}
	summary.ArchivistBackend = archivistBackendName
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	cfg, err := config.Load(abs)
	if err != nil {
		fmt.Fprintf(stderr, "learn config failed: %v\n", err)
		return 1
	}
	autoApply := cfg.Learn.AutoApply
	autoCleanupDeferred := cfg.Learn.AutoCleanupDeferred
	if autoApplyOverride != nil {
		autoApply = *autoApplyOverride
	}
	if autoCleanupDeferredOverride != nil {
		autoCleanupDeferred = *autoCleanupDeferredOverride
	}
	summary.AutoApply = autoApply
	summary.AutoCleanupDeferred = autoCleanupDeferred

	runResult, err := runner.Run(abs, runner.Options{BackendName: backendName, Task: task})
	if err != nil {
		fmt.Fprintf(stderr, "learn run failed: %v\n", err)
		return 1
	}
	summary.RunID = runResult.RunID
	summary.RunDir = runResult.RunDir
	summary.ExitCode = runResult.ExitCode
	fmt.Fprintf(humanOut, "Run %s complete: %s\n", runResult.RunID, runResult.RunDir)
	fmt.Fprintf(humanOut, "Exit code: %d\n", runResult.ExitCode)
	if runResult.ExitCode != 0 {
		summary.Status = "failed"
		summary.Outcome = "run_failed"
		summary.StoppedReason = "run failed; no proposal, archivist decision, or apply attempted"
		fmt.Fprintln(humanOut, "Learn stopped: run failed; no proposal, archivist decision, or apply attempted")
		writeLearnSummary(stdout, summary, jsonOutput)
		return 1
	}

	proposalResult, err := proposal.Generate(abs, proposal.Options{FromRun: runResult.RunID})
	if err != nil {
		fmt.Fprintf(stderr, "learn propose failed: %v\n", err)
		return 1
	}
	proposalID := filepath.Base(strings.TrimSuffix(proposalResult.Path, ".yaml"))
	summary.ProposalPath = proposalResult.Path
	summary.ProposalID = proposalID
	summary.Candidates = proposalResult.Candidates
	summary.CandidateSource = proposalResult.Source
	summary.Safety = proposalResult.Safety
	fmt.Fprintf(humanOut, "Proposal written: %s\n", proposalResult.Path)
	fmt.Fprintf(humanOut, "Candidates: %d\n", proposalResult.Candidates)
	if proposalResult.Source != "" {
		fmt.Fprintf(humanOut, "Candidate source: %s\n", proposalResult.Source)
	}
	fmt.Fprintf(humanOut, "Safety: %s\n", proposalResult.Safety)
	if proposalResult.Candidates == 0 && proposalResult.Source != "structured" {
		summary.Status = "stopped"
		summary.Outcome = "no_structured_learning"
		summary.StoppedReason = "no structured learning candidates or lifecycle operation; no fallback proposal or archivist review attempted"
		fmt.Fprintln(humanOut, "Learn stopped: no structured learning candidates or lifecycle operation; no fallback proposal or archivist review attempted")
		updated, code := cleanupDeferredIfRequested(abs, autoCleanupDeferred, humanOut, stderr)
		summary.CleanupUpdated = updated
		writeLearnSummary(stdout, summary, jsonOutput)
		if code != 0 {
			return code
		}
		return 0
	}
	if proposalResult.Source != "structured" {
		summary.Status = "stopped"
		summary.Outcome = "unsupported_candidate_source"
		summary.StoppedReason = fmt.Sprintf("candidate source is %s; no archivist review or apply attempted", proposalResult.Source)
		fmt.Fprintf(humanOut, "Learn stopped: candidate source is %s; no archivist review or apply attempted\n", proposalResult.Source)
		updated, code := cleanupDeferredIfRequested(abs, autoCleanupDeferred, humanOut, stderr)
		summary.CleanupUpdated = updated
		writeLearnSummary(stdout, summary, jsonOutput)
		if code != 0 {
			return code
		}
		return 0
	}

	fmt.Fprintf(humanOut, "Archivist backend: %s\n", archivistBackendName)
	archivistResult, err := archivist.Decide(abs, archivist.Options{Target: proposalID, BackendName: archivistBackendName})
	if err != nil {
		fmt.Fprintf(stderr, "learn archivist failed: %v\n", err)
		return 1
	}
	summary.ArchivistDecision = archivistResult.Decision
	fmt.Fprintf(humanOut, "Archivist decision: %s\n", archivistResult.Decision)
	for _, note := range archivistResult.Notes {
		fmt.Fprintf(humanOut, "  - %s\n", note)
	}
	if archivistResult.Decision != "apply" {
		summary.Status = "stopped"
		summary.Outcome = "archivist_" + archivistResult.Decision
		summary.StoppedReason = fmt.Sprintf("archivist decision is %s; no apply attempted", archivistResult.Decision)
		fmt.Fprintf(humanOut, "Learn stopped: archivist decision is %s; no apply attempted\n", archivistResult.Decision)
		updated, code := cleanupDeferredIfRequested(abs, autoCleanupDeferred, humanOut, stderr)
		summary.CleanupUpdated = updated
		writeLearnSummary(stdout, summary, jsonOutput)
		if code != 0 {
			return code
		}
		return 0
	}

	ready, err := candidates.ApplyReady(abs)
	if err != nil {
		fmt.Fprintf(stderr, "learn apply-ready check failed: %v\n", err)
		return 1
	}
	readyItem, ok := findCandidateItem(ready, proposalID)
	if !ok {
		summary.Status = "failed"
		summary.Outcome = "apply_ready_failed"
		summary.StoppedReason = "archivist approved proposal but apply-ready gates did not pass; no apply attempted"
		fmt.Fprintln(humanOut, "Learn stopped: archivist approved proposal but apply-ready gates did not pass; no apply attempted")
		writeLearnSummary(stdout, summary, jsonOutput)
		return 1
	}
	summary.ApplyReady = true
	fmt.Fprintln(humanOut, "Learn auto-apply: proposal passed apply-ready gates")
	if !autoApply {
		summary.Status = "stopped"
		summary.Outcome = "auto_apply_disabled"
		summary.StoppedReason = "auto_apply disabled; proposal remains pending"
		fmt.Fprintf(humanOut, "Learn stopped: auto_apply disabled; proposal remains pending at %s\n", readyItem.Path)
		writeLearnSummary(stdout, summary, jsonOutput)
		return 0
	}
	applyResult, err := apply.Apply(abs, apply.Options{Target: readyItem.Path})
	if err != nil {
		fmt.Fprintf(stderr, "learn apply failed: %v\n", err)
		return 1
	}
	summary.AppliedProposal = applyResult.ProposalPath
	summary.AppliedPaths = applyResult.Applied
	summary.Status = "completed"
	summary.Outcome = "applied"
	fmt.Fprintf(humanOut, "Applied proposal: %s\n", applyResult.ProposalPath)
	for _, path := range applyResult.Applied {
		fmt.Fprintf(humanOut, "  - %s\n", path)
	}
	fmt.Fprintln(humanOut, "Learn complete: applied safe approved knowledge")
	writeLearnSummary(stdout, summary, jsonOutput)
	return 0
}

type learnSummary struct {
	Status              string   `json:"status"`
	Outcome             string   `json:"outcome"`
	Task                string   `json:"task,omitempty"`
	Backend             string   `json:"backend,omitempty"`
	ArchivistBackend    string   `json:"archivist_backend,omitempty"`
	AutoApply           bool     `json:"auto_apply"`
	AutoCleanupDeferred bool     `json:"auto_cleanup_deferred"`
	RunID               string   `json:"run_id,omitempty"`
	RunDir              string   `json:"run_dir,omitempty"`
	ExitCode            int      `json:"exit_code"`
	ProposalID          string   `json:"proposal_id,omitempty"`
	ProposalPath        string   `json:"proposal_path,omitempty"`
	Candidates          int      `json:"candidates"`
	CandidateSource     string   `json:"candidate_source,omitempty"`
	Safety              string   `json:"safety,omitempty"`
	ArchivistDecision   string   `json:"archivist_decision,omitempty"`
	ApplyReady          bool     `json:"apply_ready"`
	AppliedProposal     string   `json:"applied_proposal,omitempty"`
	AppliedPaths        []string `json:"applied_paths,omitempty"`
	CleanupUpdated      []string `json:"cleanup_updated,omitempty"`
	StoppedReason       string   `json:"stopped_reason,omitempty"`
}

func writeLearnSummary(w io.Writer, summary learnSummary, jsonOutput bool) {
	if jsonOutput {
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			fmt.Fprintf(w, "{\"status\":\"failed\",\"outcome\":\"json_encode_failed\",\"stopped_reason\":%q}\n", err.Error())
			return
		}
		fmt.Fprintf(w, "%s\n", data)
		return
	}
	fmt.Fprintln(w, "Learn summary:")
	fmt.Fprintf(w, "  status: %s\n", summary.Status)
	fmt.Fprintf(w, "  outcome: %s\n", summary.Outcome)
	fmt.Fprintf(w, "  run: %s\n", summary.RunID)
	if summary.ProposalPath != "" {
		fmt.Fprintf(w, "  proposal: %s\n", summary.ProposalPath)
	}
	fmt.Fprintf(w, "  candidates: %d\n", summary.Candidates)
	if summary.ArchivistDecision != "" {
		fmt.Fprintf(w, "  archivist_decision: %s\n", summary.ArchivistDecision)
	}
	fmt.Fprintf(w, "  apply_ready: %t\n", summary.ApplyReady)
	fmt.Fprintf(w, "  applied: %d\n", len(summary.AppliedPaths))
	if len(summary.CleanupUpdated) > 0 {
		fmt.Fprintf(w, "  cleanup_updated: %d\n", len(summary.CleanupUpdated))
	}
	if summary.StoppedReason != "" {
		fmt.Fprintf(w, "  stopped_reason: %s\n", summary.StoppedReason)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func findCandidateItem(items []candidates.Item, id string) (candidates.Item, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return candidates.Item{}, false
}

func cleanupDeferredIfRequested(projectRoot string, enabled bool, stdout, stderr io.Writer) ([]string, int) {
	if !enabled {
		return nil, 0
	}
	result, err := candidates.CleanupDeferred(projectRoot, candidates.CleanupOptions{})
	if err != nil {
		fmt.Fprintf(stderr, "learn deferred cleanup failed: %v\n", err)
		return nil, 1
	}
	fmt.Fprintf(stdout, "Deferred cleanup: %d\n", len(result.Updated))
	for _, path := range result.Updated {
		fmt.Fprintf(stdout, "  - %s\n", path)
	}
	return result.Updated, 0
}

func runRemember(args []string, stdout, stderr io.Writer) int {
	root := "."
	source := "user"
	memoryType := "invariant"
	title := ""
	trigger := ""
	summaryText := ""
	backendName := ""
	archivistBackendName := ""
	jsonOutput := false
	var autoApplyOverride *bool
	var knowledgeParts []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		case "--source":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--source requires a value")
				return 2
			}
			i++
			source = args[i]
		case "--type":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--type requires a value")
				return 2
			}
			i++
			memoryType = args[i]
		case "--title":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--title requires a value")
				return 2
			}
			i++
			title = args[i]
		case "--trigger":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--trigger requires a value")
				return 2
			}
			i++
			trigger = args[i]
		case "--summary":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--summary requires a value")
				return 2
			}
			i++
			summaryText = args[i]
		case "--backend":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--backend requires a value")
				return 2
			}
			i++
			backendName = args[i]
		case "--archivist-backend", "--reviewer":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "%s requires a value\n", arg)
				return 2
			}
			i++
			archivistBackendName = args[i]
		case "--auto-apply":
			autoApplyOverride = boolPtr(true)
		case "--no-auto-apply":
			autoApplyOverride = boolPtr(false)
		case "--json":
			jsonOutput = true
		case "--help", "-h":
			fmt.Fprintln(stdout, "usage: sima remember <knowledge> [--source <user|review|agent>] [--type <memory-type>] [--title <title>] [--trigger <trigger>] [--backend <name>] [--path <path>]")
			return 0
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown option: %s\n", arg)
				return 2
			}
			knowledgeParts = append(knowledgeParts, arg)
		}
	}
	knowledge := strings.TrimSpace(strings.Join(knowledgeParts, " "))
	if knowledge == "" {
		fmt.Fprintln(stderr, "usage: sima remember <knowledge> [--source <user|review|agent>] [--type <memory-type>] [--title <title>] [--trigger <trigger>] [--backend <name>] [--path <path>]")
		return 2
	}
	humanOut := stdout
	if jsonOutput {
		humanOut = io.Discard
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	cfg, err := config.Load(abs)
	if err != nil {
		fmt.Fprintf(stderr, "remember config failed: %v\n", err)
		return 1
	}
	autoApply := cfg.Learn.AutoApply
	if autoApplyOverride != nil {
		autoApply = *autoApplyOverride
	}
	result, err := proposal.Remember(abs, proposal.RememberOptions{Text: knowledge, Source: source, Type: memoryType, Title: title, Trigger: trigger, Summary: summaryText})
	if err != nil {
		fmt.Fprintf(stderr, "remember failed: %v\n", err)
		return 1
	}
	proposalID := result.ID
	summary := learnSummary{Status: "stopped", Outcome: "candidate_created", Task: knowledge, Backend: backendName, AutoApply: autoApply, ProposalPath: result.Path, ProposalID: proposalID, Candidates: result.Candidates, CandidateSource: result.Source, Safety: result.Safety}
	fmt.Fprintf(humanOut, "Remember proposal written: %s\n", result.Path)
	fmt.Fprintf(humanOut, "Candidates: %d\n", result.Candidates)
	if backendName == "" && archivistBackendName == "" {
		summary.StoppedReason = "candidate created; pass --backend or --archivist-backend to run clean archivist/apply flow"
		fmt.Fprintf(humanOut, "Next: sima archivist --proposal %s --path %s\n", proposalID, abs)
		writeLearnSummary(stdout, summary, jsonOutput)
		return 0
	}
	if archivistBackendName == "" {
		archivistBackendName = backendName
	}
	summary.ArchivistBackend = archivistBackendName
	fmt.Fprintf(humanOut, "Archivist backend: %s\n", archivistBackendName)
	archivistResult, err := archivist.Decide(abs, archivist.Options{Target: proposalID, BackendName: archivistBackendName})
	if err != nil {
		fmt.Fprintf(stderr, "remember archivist failed: %v\n", err)
		return 1
	}
	summary.ArchivistDecision = archivistResult.Decision
	fmt.Fprintf(humanOut, "Archivist decision: %s\n", archivistResult.Decision)
	for _, note := range archivistResult.Notes {
		fmt.Fprintf(humanOut, "  - %s\n", note)
	}
	if archivistResult.Decision != "apply" {
		summary.Outcome = "archivist_" + archivistResult.Decision
		summary.StoppedReason = fmt.Sprintf("archivist decision is %s; no apply attempted", archivistResult.Decision)
		writeLearnSummary(stdout, summary, jsonOutput)
		return 0
	}
	ready, err := candidates.ApplyReady(abs)
	if err != nil {
		fmt.Fprintf(stderr, "remember apply-ready check failed: %v\n", err)
		return 1
	}
	readyItem, ok := findCandidateItem(ready, proposalID)
	if !ok {
		summary.Status = "failed"
		summary.Outcome = "apply_ready_failed"
		summary.StoppedReason = "archivist approved proposal but apply-ready gates did not pass; no apply attempted"
		writeLearnSummary(stdout, summary, jsonOutput)
		return 1
	}
	summary.ApplyReady = true
	if !autoApply {
		summary.Outcome = "auto_apply_disabled"
		summary.StoppedReason = "auto_apply disabled; proposal remains pending"
		fmt.Fprintf(humanOut, "Remember stopped: auto_apply disabled; proposal remains pending at %s\n", readyItem.Path)
		writeLearnSummary(stdout, summary, jsonOutput)
		return 0
	}
	applyResult, err := apply.Apply(abs, apply.Options{Target: readyItem.Path})
	if err != nil {
		fmt.Fprintf(stderr, "remember apply failed: %v\n", err)
		return 1
	}
	summary.AppliedProposal = applyResult.ProposalPath
	summary.AppliedPaths = applyResult.Applied
	summary.Status = "completed"
	summary.Outcome = "applied"
	fmt.Fprintf(humanOut, "Applied proposal: %s\n", applyResult.ProposalPath)
	for _, path := range applyResult.Applied {
		fmt.Fprintf(humanOut, "  - %s\n", path)
	}
	fmt.Fprintln(humanOut, "Remember complete: explicit knowledge applied through SIMA harness")
	writeLearnSummary(stdout, summary, jsonOutput)
	return 0
}

func runPropose(args []string, stdout, stderr io.Writer) int {
	root := "."
	fromRun := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--from-run":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--from-run requires a value")
				return 2
			}
			i++
			fromRun = args[i]
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", arg)
			return 2
		}
	}
	if fromRun == "" {
		fmt.Fprintln(stderr, "usage: sima propose --from-run <run-id|last|path> [--path <path>]")
		return 2
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	result, err := proposal.Generate(abs, proposal.Options{FromRun: fromRun})
	if err != nil {
		fmt.Fprintf(stderr, "propose failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Proposal written: %s\n", result.Path)
	fmt.Fprintf(stdout, "Run: %s\n", result.RunID)
	fmt.Fprintf(stdout, "Archivist decision: %s\n", result.Decision)
	fmt.Fprintf(stdout, "Safety: %s\n", result.Safety)
	fmt.Fprintf(stdout, "Candidates: %d\n", result.Candidates)
	if result.Source != "" {
		fmt.Fprintf(stdout, "Candidate source: %s\n", result.Source)
	}
	return 0
}

func runReview(args []string, stdout, stderr io.Writer) int {
	root := "."
	all := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		case "--all":
			all = true
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", arg)
			return 2
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	result, err := review.Review(abs, review.Options{All: all})
	if err != nil {
		fmt.Fprintf(stderr, "review failed: %v\n", err)
		return 1
	}
	if len(result.Items) == 0 {
		fmt.Fprintln(stdout, "No candidate proposals found")
		return 0
	}
	valid := 0
	blocked := 0
	for _, item := range result.Items {
		state := "valid"
		if len(item.Problems) > 0 || item.Safety != "safe" {
			state = "blocked"
			blocked++
		} else {
			valid++
		}
		fmt.Fprintf(stdout, "%s	%s	status=%s	decision=%s	safety=%s	destination=%s	operation=%s	candidates=%d	evidence=%d	%s\n", state, item.ID, item.Status, item.Decision, item.Safety, item.Destination, item.Operation, item.Candidates, item.Evidence, item.Path)
		for _, problem := range item.Problems {
			fmt.Fprintf(stdout, "  - %s\n", problem)
		}
	}
	fmt.Fprintf(stdout, "Summary: %d total, %d valid, %d blocked\n", len(result.Items), valid, blocked)
	return 0
}

func runApply(args []string, stdout, stderr io.Writer) int {
	root := "."
	target := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		default:
			if strings.HasPrefix(arg, "--") {
				fmt.Fprintf(stderr, "unknown option: %s\n", arg)
				return 2
			}
			if target != "" {
				fmt.Fprintln(stderr, "usage: sima apply <proposal-id|path> [--path <path>]")
				return 2
			}
			target = arg
		}
	}
	if target == "" {
		fmt.Fprintln(stderr, "usage: sima apply <proposal-id|path> [--path <path>]")
		return 2
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	result, err := apply.Apply(abs, apply.Options{Target: target})
	if err != nil {
		fmt.Fprintf(stderr, "apply failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Applied proposal: %s\n", result.ProposalPath)
	for _, path := range result.Applied {
		fmt.Fprintf(stdout, "  - %s\n", path)
	}
	return 0
}

func runArchivist(args []string, stdout, stderr io.Writer) int {
	root := "."
	target := ""
	backendName := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--proposal":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--proposal requires a value")
				return 2
			}
			i++
			target = args[i]
		case "--backend":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--backend requires a value")
				return 2
			}
			i++
			backendName = args[i]
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", arg)
			return 2
		}
	}
	if target == "" {
		fmt.Fprintln(stderr, "usage: sima archivist --proposal <proposal-id|path> [--backend <backend>] [--path <path>]")
		return 2
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	result, err := archivist.Decide(abs, archivist.Options{Target: target, BackendName: backendName})
	if err != nil {
		fmt.Fprintf(stderr, "archivist failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Archivist decision: %s\n", result.Decision)
	fmt.Fprintf(stdout, "Proposal: %s\n", result.ProposalPath)
	for _, note := range result.Notes {
		fmt.Fprintf(stdout, "  - %s\n", note)
	}
	return 0
}

func runBackend(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printBackendHelp(stderr)
		return 2
	}
	switch args[0] {
	case "list":
		return runBackendList(args[1:], stdout, stderr)
	case "add":
		return runBackendAdd(args[1:], stdout, stderr)
	case "doctor":
		return runBackendDoctor(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown backend command: %s\n\n", args[0])
		printBackendHelp(stderr)
		return 2
	}
}

func printBackendHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  sima backend list [path]
  sima backend add <name> --kind <claude-code|codex> --executable <path> [--config <path>] [--env-file <path>] [--working-dir <path>] [--permission-mode <mode>] [--metadata KEY=VALUE] [--force]
  sima backend doctor <name> [path]`)
}

func runBackendList(args []string, stdout, stderr io.Writer) int {
	root := "."
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: sima backend list [path]")
		return 2
	}
	if len(args) == 1 {
		root = args[0]
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	cfg, err := config.Load(abs)
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}
	if len(cfg.Backends) == 0 {
		fmt.Fprintln(stdout, "No backends configured")
		return 0
	}
	names := make([]string, 0, len(cfg.Backends))
	for name := range cfg.Backends {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		profile := cfg.Backends[name]
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", name, profile.Kind, profile.Executable)
	}
	return 0
}

func runBackendAdd(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: sima backend add <name> --kind <claude-code|codex> --executable <path> [options]")
		return 2
	}
	name := args[0]
	root := "."
	profile := config.BackendProfile{Env: map[string]string{}}
	force := false

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--kind":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--kind requires a value")
				return 2
			}
			i++
			profile.Kind = args[i]
		case "--executable":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--executable requires a value")
				return 2
			}
			i++
			profile.Executable = args[i]
		case "--config":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--config requires a value")
				return 2
			}
			i++
			profile.ConfigPath = args[i]
		case "--env-file":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--env-file requires a value")
				return 2
			}
			i++
			profile.EnvFile = args[i]
		case "--working-dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--working-dir requires a value")
				return 2
			}
			i++
			profile.WorkingDir = args[i]
		case "--permission-mode":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--permission-mode requires a value")
				return 2
			}
			i++
			profile.PermissionMode = args[i]
		case "--env":
			if i+1 >= len(args) || !strings.Contains(args[i+1], "=") {
				fmt.Fprintln(stderr, "--env requires KEY=VALUE")
				return 2
			}
			i++
			parts := strings.SplitN(args[i], "=", 2)
			profile.Env[parts[0]] = parts[1]
		case "--metadata":
			if i+1 >= len(args) || !strings.Contains(args[i+1], "=") {
				fmt.Fprintln(stderr, "--metadata requires KEY=VALUE")
				return 2
			}
			i++
			if profile.Metadata == nil {
				profile.Metadata = map[string]string{}
			}
			parts := strings.SplitN(args[i], "=", 2)
			profile.Metadata[parts[0]] = parts[1]
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		case "--force":
			force = true
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", arg)
			return 2
		}
	}
	if len(profile.Env) == 0 {
		profile.Env = nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	if err := config.AddBackend(abs, name, profile, force); err != nil {
		fmt.Fprintf(stderr, "add backend: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Added backend %q (%s)\n", name, profile.Kind)
	return 0
}

func runCandidates(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: sima candidates <list|apply-ready|show|cleanup> [options]")
		return 2
	}
	switch args[0] {
	case "list":
		return runCandidatesList(args[1:], stdout, stderr)
	case "apply-ready":
		return runCandidatesApplyReady(args[1:], stdout, stderr)
	case "show":
		return runCandidatesShow(args[1:], stdout, stderr)
	case "cleanup":
		return runCandidatesCleanup(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown candidates command: %s\n", args[0])
		return 2
	}
}

func runCandidatesList(args []string, stdout, stderr io.Writer) int {
	root := "."
	status := "candidate"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		case "--status":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--status requires a value")
				return 2
			}
			i++
			status = args[i]
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", args[i])
			return 2
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	items, err := candidates.List(abs, candidates.ListOptions{Status: status})
	if err != nil {
		fmt.Fprintf(stderr, "candidate list failed: %v\n", err)
		return 1
	}
	printCandidateItems(stdout, items)
	return 0
}

func runCandidatesApplyReady(args []string, stdout, stderr io.Writer) int {
	root := "."
	applyReady := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		case "--apply":
			applyReady = true
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", args[i])
			return 2
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	items, err := candidates.ApplyReady(abs)
	if err != nil {
		fmt.Fprintf(stderr, "candidate apply-ready failed: %v\n", err)
		return 1
	}
	if !applyReady {
		printCandidateItems(stdout, items)
		return 0
	}
	fmt.Fprintf(stdout, "Applying candidates: %d\n", len(items))
	for _, item := range items {
		result, err := apply.Apply(abs, apply.Options{Target: item.Path})
		if err != nil {
			fmt.Fprintf(stderr, "apply-ready failed for %s: %v\n", item.Path, err)
			return 1
		}
		fmt.Fprintf(stdout, "  - %s\n", result.ProposalPath)
		for _, path := range result.Applied {
			fmt.Fprintf(stdout, "    applied: %s\n", path)
		}
	}
	return 0
}

func printCandidateItems(stdout io.Writer, items []candidates.Item) {
	fmt.Fprintln(stdout, "STATUS	DECISION	SAFETY	DESTINATION	OPERATION	CANDIDATES	ID	PATH")
	for _, item := range items {
		fmt.Fprintf(stdout, "%s	%s	%s	%s	%s	%d	%s	%s\n", item.Status, item.Decision, item.Safety, item.Destination, item.Operation, item.Candidates, item.ID, item.Path)
	}
}

func runCandidatesShow(args []string, stdout, stderr io.Writer) int {
	root := "."
	target := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		default:
			if target != "" {
				fmt.Fprintln(stderr, "usage: sima candidates show <id|path> [--path <path>]")
				return 2
			}
			target = args[i]
		}
	}
	if target == "" {
		fmt.Fprintln(stderr, "usage: sima candidates show <id|path> [--path <path>]")
		return 2
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	result, err := candidates.Show(abs, target)
	if err != nil {
		fmt.Fprintf(stderr, "candidate show failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Candidate: %s\n", result.Path)
	fmt.Fprintf(stdout, "ID: %s\nStatus: %s\nDecision: %s\nSafety: %s\nDestination: %s\nOperation: %s\nCandidates: %d\n\n", result.Proposal.ID, result.Proposal.Status, result.Proposal.ArchivistDecision, result.Proposal.Safety.Decision, result.Proposal.Learning.Destination, result.Proposal.Learning.Operation, len(result.Proposal.CandidateMemories)+len(result.Proposal.CandidateSkills))
	fmt.Fprint(stdout, result.Content)
	if !strings.HasSuffix(result.Content, "\n") {
		fmt.Fprintln(stdout)
	}
	return 0
}

func runCandidatesCleanup(args []string, stdout, stderr io.Writer) int {
	root := "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", args[i])
			return 2
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	result, err := candidates.CleanupDeferred(abs, candidates.CleanupOptions{})
	if err != nil {
		fmt.Fprintf(stderr, "candidate cleanup failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Cleaned candidates: %d\n", len(result.Updated))
	for _, path := range result.Updated {
		fmt.Fprintf(stdout, "  - %s\n", path)
	}
	return 0
}

func runCatalogCommand(kind string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "list" {
		fmt.Fprintf(stderr, "usage: sima %s list [--status <active|deprecated|superseded|archived|all>] [--path <path>]\n", kind)
		return 2
	}
	root := "."
	status := "active"
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--path requires a value")
				return 2
			}
			i++
			root = args[i]
		case "--status":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--status requires a value")
				return 2
			}
			i++
			status = args[i]
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", args[i])
			return 2
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	items, err := catalog.List(abs, catalog.Options{Kind: kind, Status: status})
	if err != nil {
		fmt.Fprintf(stderr, "%s list failed: %v\n", kind, err)
		return 1
	}
	fmt.Fprintln(stdout, "STATUS\tSCOPE\tKIND\tTITLE\tPATH")
	for _, item := range items {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n", item.Status, item.Scope, item.Kind, item.Title, item.Path)
	}
	return 0
}

func runBackendDoctor(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintln(stderr, "usage: sima backend doctor <name> [path]")
		return 2
	}
	name := args[0]
	root := "."
	if len(args) == 2 {
		root = args[1]
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}
	cfg, err := config.Load(abs)
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}
	profile, ok := cfg.Backends[name]
	if !ok {
		fmt.Fprintf(stderr, "backend %q not found\n", name)
		return 1
	}
	result := backend.Doctor(name, profile)
	mark := "ok"
	if !result.OK {
		mark = "fail"
	}
	fmt.Fprintf(stdout, "[%s] %s: %s\n", mark, result.Name, result.Detail)
	if !result.OK {
		return 1
	}
	return 0
}
