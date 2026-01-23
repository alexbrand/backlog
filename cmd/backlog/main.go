package main

import (
	"os"

	"github.com/alexbrand/backlog/internal/cli"
	"github.com/alexbrand/backlog/internal/github"
	"github.com/alexbrand/backlog/internal/linear"
	"github.com/alexbrand/backlog/internal/local"
)

func init() {
	// Register all backends
	local.Register()
	github.Register()
	linear.Register()
}

func main() {
	if err := cli.Execute(); err != nil {
		// JSON format errors go to stdout for parseability,
		// all other error formats go to stderr
		format := cli.GetFormat()
		if format == "json" {
			cli.PrintError(os.Stdout, err, format)
		} else {
			cli.PrintError(os.Stderr, err, format)
		}
		os.Exit(cli.GetExitCode(err))
	}
}
