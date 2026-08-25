package cli

import (
	"fmt"
	"io"
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

const Version = "0.1.0-dev"

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
  sima doctor [path]
  sima lint [path]
  sima brief <task> [--path <path>]
  sima run --backend <name> --task <task> [--path <path>] [--no-propose]
  sima learn --backend <name> --task <task> [--archivist-backend <name>] [--path <path>]
  sima propose --from-run <run-id|last|path> [--path <path>]
  sima review [--path <path>] [--all]
  sima apply <proposal-id|path> [--path <path>]
  sima archivist --proposal <proposal-id|path> [--backend <backend>] [--path <path>]
  sima candidates list [--status <candidate|deferred|applied|rejected|all>] [--path <path>]
  sima candidates apply-ready [--path <path>]
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
  doctor   Check SIMA project state and local runtime prerequisites
  lint     Check SIMA knowledge lifecycle metadata, candidates, and target paths
  brief    Create a compact task briefing from SIMA memory, skills, and SDD artifacts
  run      Run a bounded task through a named backend, capture artifacts, and auto-propose candidates
  learn    Run the full gated self-improvement loop: run, propose, archivist, apply
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
	for _, check := range report.Checks {
		mark := "ok"
		if !check.OK {
			mark = "fail"
		}
		fmt.Fprintf(stdout, "[%s] %s", mark, check.Name)
		if check.Detail != "" {
			fmt.Fprintf(stdout, ": %s", check.Detail)
		}
		fmt.Fprintln(stdout)
	}
	if !report.OK() {
		fmt.Fprintln(stderr, "SIMA doctor found problems")
		return 1
	}
	return 0
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
		default:
			fmt.Fprintf(stderr, "unknown option: %s\n", arg)
			return 2
		}
	}
	if backendName == "" || task == "" {
		fmt.Fprintln(stderr, "usage: sima learn --backend <name> --task <task> [--archivist-backend <name>] [--path <path>]")
		return 2
	}
	if archivistBackendName == "" {
		archivistBackendName = backendName
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}

	runResult, err := runner.Run(abs, runner.Options{BackendName: backendName, Task: task})
	if err != nil {
		fmt.Fprintf(stderr, "learn run failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Run %s complete: %s\n", runResult.RunID, runResult.RunDir)
	fmt.Fprintf(stdout, "Exit code: %d\n", runResult.ExitCode)
	if runResult.ExitCode != 0 {
		fmt.Fprintln(stdout, "Learn stopped: run failed; no proposal, archivist decision, or apply attempted")
		return 1
	}

	proposalResult, err := proposal.Generate(abs, proposal.Options{FromRun: runResult.RunID})
	if err != nil {
		fmt.Fprintf(stderr, "learn propose failed: %v\n", err)
		return 1
	}
	proposalID := filepath.Base(strings.TrimSuffix(proposalResult.Path, ".yaml"))
	fmt.Fprintf(stdout, "Proposal written: %s\n", proposalResult.Path)
	fmt.Fprintf(stdout, "Candidates: %d\n", proposalResult.Candidates)
	if proposalResult.Source != "" {
		fmt.Fprintf(stdout, "Candidate source: %s\n", proposalResult.Source)
	}
	fmt.Fprintf(stdout, "Safety: %s\n", proposalResult.Safety)
	if proposalResult.Candidates == 0 && proposalResult.Source != "structured" {
		fmt.Fprintln(stdout, "Learn stopped: no structured learning candidates or lifecycle operation; no fallback proposal or archivist review attempted")
		return 0
	}
	if proposalResult.Source != "structured" {
		fmt.Fprintf(stdout, "Learn stopped: candidate source is %s; no archivist review or apply attempted\n", proposalResult.Source)
		return 0
	}

	fmt.Fprintf(stdout, "Archivist backend: %s\n", archivistBackendName)
	archivistResult, err := archivist.Decide(abs, archivist.Options{Target: proposalID, BackendName: archivistBackendName})
	if err != nil {
		fmt.Fprintf(stderr, "learn archivist failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Archivist decision: %s\n", archivistResult.Decision)
	for _, note := range archivistResult.Notes {
		fmt.Fprintf(stdout, "  - %s\n", note)
	}
	if archivistResult.Decision != "apply" {
		fmt.Fprintf(stdout, "Learn stopped: archivist decision is %s; no apply attempted\n", archivistResult.Decision)
		return 0
	}

	applyResult, err := apply.Apply(abs, apply.Options{Target: proposalID})
	if err != nil {
		fmt.Fprintf(stderr, "learn apply failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Applied proposal: %s\n", applyResult.ProposalPath)
	for _, path := range applyResult.Applied {
		fmt.Fprintf(stdout, "  - %s\n", path)
	}
	fmt.Fprintln(stdout, "Learn complete: applied safe approved knowledge")
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
		fmt.Fprintln(stderr, "usage: sima candidates <list|show|cleanup> [options]")
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
	items, err := candidates.ApplyReady(abs)
	if err != nil {
		fmt.Fprintf(stderr, "candidate apply-ready failed: %v\n", err)
		return 1
	}
	printCandidateItems(stdout, items)
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
