package cli

import (
	"fmt"
	"io"
	"path/filepath"

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
  sima version

Current v0 slice:
  init    Create project-local .sima scaffold
  doctor  Check SIMA project state and local runtime prerequisites`)
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
