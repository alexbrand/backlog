package main

import (
	"os"

	"github.com/alexbrand/backlog/internal/cli"
	"github.com/alexbrand/backlog/internal/local"
)

func init() {
	// Register all backends
	local.Register()
}

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(cli.GetExitCode(err))
	}
}
