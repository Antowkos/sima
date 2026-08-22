package main

import (
	"os"

	"github.com/antowkos/sima/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args, os.Stdout, os.Stderr))
}
