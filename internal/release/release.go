// Package release coordinates version, changelog, release commit, and tag creation.
package release

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/savutro/go-commit-tooling/internal/changelog"
	"github.com/savutro/go-commit-tooling/internal/commit"
	"github.com/savutro/go-commit-tooling/internal/gitutil"
	"github.com/savutro/go-commit-tooling/internal/tui"
	"github.com/savutro/go-commit-tooling/internal/version"
)

// Options configures an interactive release.
type Options struct {
	Input      io.Reader
	Output     io.Writer
	StrictOnly bool
}

type analysis struct {
	Suggested version.BumpKind
	Breaking  []string
}

// Run creates a release commit and annotated tag from the current Git history.
func Run(opts Options) error {
	status, err := gitutil.DirtyStatus()
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("working tree must be clean before releasing:\n%s", status)
	}

	current, err := version.Read("VERSION")
	if err != nil {
		return err
	}
	releaseAnalysis, err := analyze(opts.StrictOnly)
	if err != nil {
		return err
	}
	if len(releaseAnalysis.Breaking) > 0 {
		fmt.Fprintf(opts.Output, "Warning: %d breaking commit(s) found since the last release.\n", len(releaseAnalysis.Breaking))
		for _, item := range releaseAnalysis.Breaking {
			fmt.Fprintf(opts.Output, "- %s\n", item)
		}
		fmt.Fprintln(opts.Output, "A major version bump is recommended.")
		fmt.Fprintln(opts.Output)
	}

	next, err := version.Prompt(opts.Input, opts.Output, current, releaseAnalysis.Suggested)
	if err != nil {
		return err
	}
	tag := "v" + next.String()
	exists, err := gitutil.TagExists(tag)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("tag %s already exists", tag)
	}

	confirmed := true
	description := fmt.Sprintf("This writes VERSION and CHANGELOG.md, commits chore(release): %s, then creates tag %s.", tag, tag)
	if len(releaseAnalysis.Breaking) > 0 {
		description += " Breaking commits were detected; verify the selected version before continuing."
	}
	if err := tui.Form(
		opts.Input,
		opts.Output,
		huh.NewGroup(
			huh.NewConfirm().
				Title("Create release?").
				Description(description).
				Affirmative("Release").
				Negative("Cancel").
				Value(&confirmed),
		),
	).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errors.New("release cancelled")
		}
		return err
	}
	if !confirmed {
		return errors.New("release cancelled")
	}

	if err := version.Write("VERSION", next); err != nil {
		return err
	}
	if err := changelog.Generate(changelog.Options{
		StrictOnly:     opts.StrictOnly,
		OutputPath:     "CHANGELOG.md",
		ReleaseVersion: next.String(),
		Stdout:         opts.Output,
	}); err != nil {
		return err
	}

	if err := gitutil.Add("VERSION", "CHANGELOG.md"); err != nil {
		return err
	}
	if err := gitutil.CommitAll("chore(release): " + tag); err != nil {
		return err
	}
	if err := gitutil.AnnotatedTag(tag, tag); err != nil {
		return err
	}

	fmt.Fprintf(opts.Output, "Created release commit and tag %s.\n", tag)
	return nil
}

func analyze(strictOnly bool) (analysis, error) {
	latest, ok, err := gitutil.LastSemverTag()
	if err != nil {
		return analysis{Suggested: version.SuggestedPatch}, err
	}
	rangeSpec := ""
	if ok {
		rangeSpec = latest.Name + "..HEAD"
	}
	commits, err := gitutil.Log(rangeSpec)
	if err != nil {
		return analysis{Suggested: version.SuggestedPatch}, err
	}

	result := analysis{Suggested: version.SuggestedPatch}
	for _, gitCommit := range commits {
		parsed, conventional := commit.Parse(gitCommit.Message)
		if strictOnly && !conventional {
			continue
		}
		if parsed.Breaking || strings.Contains(gitCommit.Message, "BREAKING CHANGE:") {
			result.Suggested = version.SuggestedMajor
			result.Breaking = append(result.Breaking, fmt.Sprintf("%s %s", gitCommit.ShortHash(), firstLine(gitCommit.Message)))
			continue
		}
		if conventional && parsed.Type == "feat" && result.Suggested != version.SuggestedMajor {
			result.Suggested = version.SuggestedMinor
		}
	}
	return result, nil
}

func firstLine(message string) string {
	line, _, _ := strings.Cut(message, "\n")
	return strings.TrimSpace(line)
}
