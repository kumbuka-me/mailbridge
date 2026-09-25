package main

import (
	"context"
	"os"

	"github.com/containeroo/mailbridge/internal/app"
)

var (
	Version = "dev"
	Commit  = "none"
)

// main runs mailbridge and exits non-zero on failure.
func main() {
	if err := app.Run(context.Background(), os.Args[1:], Version, Commit, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
