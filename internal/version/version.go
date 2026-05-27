package version

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/savutro/go-commit-tooling/internal/tui"
)

// Semver is a strict x.y.z semantic version used for release files and tags.
type Semver struct {
	Major int
	Minor int
	Patch int
}

// Parse reads a strict x.y.z version string.
func Parse(input string) (Semver, error) {
	parts := strings.Split(strings.TrimSpace(input), ".")
	if len(parts) != 3 {
		return Semver{}, fmt.Errorf("version must use x.y.z format")
	}
	var values [3]int
	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return Semver{}, fmt.Errorf("version segment %q must be a non-negative integer", part)
		}
		values[i] = value
	}
	return Semver{Major: values[0], Minor: values[1], Patch: values[2]}, nil
}

// String formats the version as x.y.z.
func (v Semver) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Bump returns the next version for a major, minor, or patch release.
func (v Semver) Bump(kind string) Semver {
	switch kind {
	case "major":
		return Semver{Major: v.Major + 1}
	case "minor":
		return Semver{Major: v.Major, Minor: v.Minor + 1}
	default:
		return Semver{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	}
}

// Read loads a version file, defaulting to 0.0.0 when it does not exist.
func Read(path string) (Semver, error) {
	current := Semver{}
	if content, err := os.ReadFile(path); err == nil {
		parsed, parseErr := Parse(string(content))
		if parseErr == nil {
			return parsed, nil
		}
		return Semver{}, parseErr
	} else if !os.IsNotExist(err) {
		return Semver{}, err
	}
	return current, nil
}

// Write stores a version file in the plain x.y.z format.
func Write(path string, v Semver) error {
	return os.WriteFile(path, []byte(v.String()+"\n"), 0644)
}

// Walk opens an interactive version bump form and writes the selected version.
func Walk(in io.Reader, out io.Writer, path string) error {
	current, err := Read(path)
	if err != nil {
		return err
	}

	next, err := Prompt(in, out, current, SuggestedPatch)
	if err != nil {
		return err
	}

	if err := Write(path, next); err != nil {
		return err
	}
	fmt.Fprintf(out, "Wrote %s with %s\n", path, next)
	return nil
}

// BumpKind identifies a semantic version increment.
type BumpKind string

const (
	// SuggestedMajor means the next release contains breaking changes.
	SuggestedMajor BumpKind = "major"
	// SuggestedMinor means the next release adds backwards-compatible behavior.
	SuggestedMinor BumpKind = "minor"
	// SuggestedPatch means the next release contains backwards-compatible fixes.
	SuggestedPatch BumpKind = "patch"
)

// Prompt asks for a semantic version bump using arrow-key choices.
func Prompt(in io.Reader, out io.Writer, current Semver, suggested BumpKind) (Semver, error) {
	kind := string(suggested)
	custom := ""
	formInput := accessibleInput(in)

	form := tui.Form(
		formInput,
		out,
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(fmt.Sprintf("Current version: %s", current)).
				Description("Choose the semantic version bump. Use custom when you need an exact version.").
				Options(
					huh.NewOption("major   Breaking API or behavior change", "major"),
					huh.NewOption("minor   Backwards-compatible feature", "minor"),
					huh.NewOption("patch   Backwards-compatible bug fix or maintenance release", "patch"),
					huh.NewOption("custom  Enter an exact x.y.z version", "custom"),
				).
				Value(&kind).
				Height(7),
		),
	)

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return Semver{}, errors.New("version update cancelled")
		}
		return Semver{}, err
	}

	next := current.Bump(kind)
	if kind != "custom" {
		return next, nil
	}

	customForm := tui.Form(
		formInput,
		out,
		huh.NewGroup(
			huh.NewInput().
				Title("Custom version").
				Description("Enter the exact version to write. Use x.y.z format.").
				Placeholder("1.2.3").
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return errors.New("custom version is required")
					}
					_, err := Parse(value)
					return err
				}).
				Value(&custom),
		),
	)

	if err := customForm.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return Semver{}, errors.New("version update cancelled")
		}
		return Semver{}, err
	}

	parsed, err := Parse(custom)
	if err != nil {
		return Semver{}, err
	}
	return parsed, nil
}

func accessibleInput(in io.Reader) io.Reader {
	if os.Getenv("TERM") != "dumb" {
		return in
	}
	return &oneByteReader{source: in}
}

type oneByteReader struct {
	source io.Reader
}

func (r *oneByteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	var b [1]byte
	n, err := r.source.Read(b[:])
	if n > 0 {
		p[0] = b[0]
		return 1, nil
	}
	return 0, err
}
