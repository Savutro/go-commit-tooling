// Package stage provides the interactive git staging workflow.
package stage

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/charmbracelet/huh"

	"github.com/savutro/go-commit-tooling/internal/gitutil"
	"github.com/savutro/go-commit-tooling/internal/tui"
)

// Options configures the interactive staging command.
type Options struct {
	Input  io.Reader
	Output io.Writer
}

// Run opens an interactive file picker and applies the selected staging state.
func Run(opts Options) error {
	entries, err := gitutil.Status()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintln(opts.Output, "Working tree is clean. Nothing to stage.")
		return nil
	}

	options := make([]huh.Option[string], 0, len(entries))
	selected := make([]string, 0, len(entries))
	initiallyStaged := map[string]bool{}
	for _, entry := range entries {
		options = append(options, huh.NewOption(entry.Label(), entry.Path))
		if entry.Staged() {
			selected = append(selected, entry.Path)
			initiallyStaged[entry.Path] = true
		}
	}

	if err := tui.Form(
		opts.Input,
		opts.Output,
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Stage files").
				Description("Space toggles a file. Checked files will be staged; unchecked staged files will be unstaged. Use / to filter.").
				Options(options...).
				Value(&selected).
				Height(16),
		),
	).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errors.New("staging cancelled")
		}
		return err
	}

	selectedSet := map[string]bool{}
	for _, path := range selected {
		selectedSet[path] = true
	}

	var toStage []string
	var toUnstage []string
	for _, entry := range entries {
		wasStaged := initiallyStaged[entry.Path]
		isSelected := selectedSet[entry.Path]
		switch {
		case isSelected && !wasStaged:
			toStage = append(toStage, entry.Path)
		case !isSelected && wasStaged:
			toUnstage = append(toUnstage, entry.Path)
		}
	}

	if err := gitutil.Add(toStage...); err != nil {
		return err
	}
	if err := gitutil.RestoreStaged(toUnstage...); err != nil {
		return err
	}

	printSummary(opts.Output, toStage, toUnstage, selected)
	return nil
}

func printSummary(out io.Writer, staged, unstaged, selected []string) {
	if len(staged) == 0 && len(unstaged) == 0 {
		fmt.Fprintln(out, "Staging unchanged.")
		return
	}
	if len(staged) > 0 {
		fmt.Fprintf(out, "Staged %d file(s): %s\n", len(staged), joinPaths(staged))
	}
	if len(unstaged) > 0 {
		fmt.Fprintf(out, "Unstaged %d file(s): %s\n", len(unstaged), joinPaths(unstaged))
	}
	fmt.Fprintf(out, "%d file(s) selected for the next commit.\n", len(selected))
}

func joinPaths(paths []string) string {
	clean := slices.Clone(paths)
	slices.Sort(clean)
	return fmt.Sprintf("%q", clean)
}
