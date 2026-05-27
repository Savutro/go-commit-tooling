// Gct manages conventional commits, changelogs, and semantic versions.
package main

import (
	"os"

	"github.com/savutro/go-commit-tooling/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
