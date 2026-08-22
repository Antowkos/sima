package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antowkos/sima/internal/backend"
	"github.com/antowkos/sima/internal/brief"
	"github.com/antowkos/sima/internal/config"
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
	case "backend":
		return runBackend(args[2:], stdout, stderr)
	case "brief":
		return runBrief(args[2:], stdout, stderr)
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
  sima brief <task> [--path <path>]
  sima backend list [path]
  sima backend add <name> --kind <claude-code|codex> --executable <path> [options]
  sima backend doctor <name> [path]
  sima version

Current v0 slice:
  init     Create project-local .sima scaffold
  doctor   Check SIMA project state and local runtime prerequisites
  brief    Create a compact task briefing from SIMA memory, skills, and SDD artifacts
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
  sima backend add <name> --kind <claude-code|codex> --executable <path> [--config <path>] [--env-file <path>] [--working-dir <path>] [--permission-mode <mode>] [--force]
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
