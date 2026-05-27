package cli

import (
	"flag"
	"fmt"
	"io"
	"os/exec"

	"github.com/charmbracelet/huh"

	"github.com/savutro/go-commit-tooling/internal/changelog"
	"github.com/savutro/go-commit-tooling/internal/commit"
	"github.com/savutro/go-commit-tooling/internal/release"
	"github.com/savutro/go-commit-tooling/internal/stage"
	"github.com/savutro/go-commit-tooling/internal/tui"
	"github.com/savutro/go-commit-tooling/internal/version"
)

const usage = `gct helps with conventional commits, changelogs, and semantic versions.

Usage:
  gct add                 Interactively stage and unstage files
  gct commit              Build and create a conventional git commit
  gct generate [-s]       Generate CHANGELOG.md from git history
  gct version             Create or update VERSION interactively
  gct release [-s]        Bump VERSION, generate changelog, commit, and tag
  gct help                Show this help
`

// Execute runs the CLI command and returns a process exit code.
func Execute(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(stdout, usage)
		return 0
	}

	var err error
	switch args[0] {
	case "add":
		err = runAdd(stdin, stdout)
	case "commit":
		err = runCommit(stdin, stdout, stderr)
	case "generate":
		err = runGenerate(args[1:], stdout)
	case "version":
		err = runVersion(stdin, stdout)
	case "release":
		err = runRelease(args[1:], stdin, stdout)
	default:
		err = fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}

	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runAdd(stdin io.Reader, stdout io.Writer) error {
	return stage.Run(stage.Options{Input: stdin, Output: stdout})
}

func runCommit(stdin io.Reader, stdout, stderr io.Writer) error {
	builder := commit.NewBuilder(stdin, stderr)
	message, err := builder.Build()
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "\nCommit message:\n%s\n\n", message)
	confirmed := true
	if err := tui.Form(
		stdin,
		stderr,
		huh.NewGroup(
			huh.NewConfirm().
				Title("Create git commit now?").
				Affirmative("Commit").
				Negative("Cancel").
				Value(&confirmed),
		),
	).Run(); err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	cmd := exec.Command("git", "commit", "-m", message)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		fmt.Fprint(stderr, string(out))
	}
	if err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	return nil
}

func runGenerate(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	strictOnly := flags.Bool("s", false, "only include conventional commits")
	output := flags.String("o", "CHANGELOG.md", "output file")
	releaseVersion := flags.String("version", "", "label the current range as this release version")
	if err := flags.Parse(args); err != nil {
		return err
	}

	return changelog.Generate(changelog.Options{
		StrictOnly:     *strictOnly,
		OutputPath:     *output,
		ReleaseVersion: *releaseVersion,
		Stdout:         stdout,
	})
}

func runVersion(stdin io.Reader, stdout io.Writer) error {
	return version.Walk(stdin, stdout, "VERSION")
}

func runRelease(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("release", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	strictOnly := flags.Bool("s", false, "only include conventional commits in the changelog")
	if err := flags.Parse(args); err != nil {
		return err
	}
	return release.Run(release.Options{
		Input:      stdin,
		Output:     stdout,
		StrictOnly: *strictOnly,
	})
}
