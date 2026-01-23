package main

import (
	"os"

	"github.com/alexbrand/backlog/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
