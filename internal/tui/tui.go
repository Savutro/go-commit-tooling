// Package tui centralizes terminal form defaults.
package tui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// Form creates a huh form that renders in the alternate screen.
func Form(in io.Reader, out io.Writer, groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithProgramOptions(tea.WithAltScreen(), tea.WithReportFocus()).
		WithInput(in).
		WithOutput(out).
		WithShowHelp(true)
}
