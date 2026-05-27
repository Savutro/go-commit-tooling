package commit

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/savutro/go-commit-tooling/internal/tui"
)

// Type describes one Conventional Commit type and when to use it.
type Type struct {
	Name        string
	Title       string
	Description string
}

// Types is the curated list shown by the interactive commit builder.
var Types = []Type{
	{Name: "feat", Title: "Feature", Description: "Adds or changes user-facing behavior or a public capability."},
	{Name: "fix", Title: "Fix", Description: "Corrects broken behavior, regressions, or user-visible bugs."},
	{Name: "docs", Title: "Documentation", Description: "Documentation-only changes such as README, guides, or examples."},
	{Name: "style", Title: "Style", Description: "Formatting, whitespace, or lint-only changes without behavior changes."},
	{Name: "refactor", Title: "Refactor", Description: "Code restructuring that neither fixes a bug nor adds a feature."},
	{Name: "perf", Title: "Performance", Description: "Improves speed, memory use, startup time, or runtime efficiency."},
	{Name: "test", Title: "Tests", Description: "Adds, updates, or fixes tests without changing production behavior."},
	{Name: "build", Title: "Build", Description: "Build system, dependencies, packaging, or generated artifact changes."},
	{Name: "ci", Title: "CI", Description: "Continuous integration, delivery, workflow, or automation changes."},
	{Name: "chore", Title: "Chore", Description: "Maintenance that does not affect source behavior, tests, docs, or builds."},
	{Name: "revert", Title: "Revert", Description: "Reverts an earlier commit. Mention the reverted commit in the body."},
}

// Builder collects the fields needed to produce a Conventional Commit message.
type Builder struct {
	in  io.Reader
	out io.Writer
}

// NewBuilder returns an interactive commit message builder.
func NewBuilder(in io.Reader, out io.Writer) Builder {
	return Builder{in: in, out: out}
}

// Build walks the user through an arrow-key driven commit message form.
func (b Builder) Build() (string, error) {
	var (
		commitType  = "feat"
		scope       string
		breaking    bool
		description string
		body        string
		footer      string
	)

	form := tui.Form(
		b.in,
		b.out,
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Commit type").
				DescriptionFunc(func() string { return typeDescription(commitType) }, &commitType).
				Options(typeOptions()...).
				Height(8).
				Value(&commitType),
			huh.NewInput().
				Title("Scope").
				Description("Optional. The affected area, package, page, or component, for example cli or docs.").
				Placeholder("cli").
				Value(&scope),
			huh.NewConfirm().
				Title("Breaking change?").
				Description("Choose yes only when users must change how they use the project after this commit.").
				Affirmative("Breaking").
				Negative("Compatible").
				Value(&breaking),
			huh.NewInput().
				Title("Short description").
				Description("Required. Use imperative mood, lower case, no trailing period: add release command.").
				Placeholder("add release command").
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return errors.New("description is required")
					}
					return nil
				}).
				Value(&description),
		),
		huh.NewGroup(
			huh.NewText().
				Title("Body").
				Description("Optional. Explain why the change was made or what tradeoffs matter.").
				Lines(5).
				ExternalEditor(true).
				Value(&body),
			huh.NewText().
				Title("Footer").
				Description("Optional. Use issue refs like Closes #123 or BREAKING CHANGE: migration details.").
				Lines(3).
				ExternalEditor(true).
				Value(&footer),
		),
	)

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", errors.New("commit cancelled")
		}
		return "", err
	}

	header := commitType
	if scope = cleanInline(scope); scope != "" {
		header += "(" + scope + ")"
	}
	if breaking {
		header += "!"
	}
	header += ": " + cleanInline(description)

	parts := []string{header}
	if body = cleanBlock(body); body != "" {
		parts = append(parts, body)
	}
	if footer = cleanBlock(footer); footer != "" {
		parts = append(parts, footer)
	}
	return strings.Join(parts, "\n\n"), nil
}

// TitleFor returns the human readable changelog heading for a commit type.
func TitleFor(name string) string {
	for _, t := range Types {
		if t.Name == name {
			return t.Title
		}
	}
	if name == "" {
		return "Other"
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func typeOptions() []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(Types))
	for _, t := range Types {
		label := fmt.Sprintf("%-8s %s", t.Name, t.Title)
		options = append(options, huh.NewOption(label, t.Name))
	}
	return options
}

func typeDescription(name string) string {
	for _, t := range Types {
		if t.Name == name {
			return t.Description
		}
	}
	return "Use arrow keys to choose the type that best describes the intent of this commit."
}

func cleanInline(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func cleanBlock(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
}
