// Package changelog turns Git history into a versioned Markdown changelog.
package changelog

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/savutro/go-commit-tooling/internal/commit"
	"github.com/savutro/go-commit-tooling/internal/gitutil"
)

// Options controls changelog generation.
type Options struct {
	StrictOnly     bool
	OutputPath     string
	ReleaseVersion string
	Stdout         io.Writer
}

type section struct {
	Version string
	Date    string
	Entries []entry
}

type entry struct {
	Hash         string
	Type         string
	Scope        string
	Description  string
	Footer       string
	Conventional bool
}

// Generate writes a changelog grouped by release tags plus the current range.
func Generate(opts Options) error {
	if opts.OutputPath == "" {
		opts.OutputPath = "CHANGELOG.md"
	}

	sections, err := sections(opts)
	if err != nil {
		return err
	}

	content := render(sections)
	if err := os.WriteFile(opts.OutputPath, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "Wrote %s with %d sections.\n", opts.OutputPath, len(sections))
	return nil
}

func sections(opts Options) ([]section, error) {
	tags, err := gitutil.SemverTags()
	if err != nil {
		return nil, err
	}

	var result []section
	currentRange := ""
	if len(tags) > 0 {
		currentRange = tags[len(tags)-1].Name + "..HEAD"
	}
	currentEntries, err := entriesForRange(currentRange, opts.StrictOnly)
	if err != nil {
		return nil, err
	}
	if len(currentEntries) > 0 || opts.ReleaseVersion != "" {
		version := "Unreleased"
		date := ""
		if opts.ReleaseVersion != "" {
			version = opts.ReleaseVersion
			date = time.Now().Format("2006-01-02")
		}
		result = append(result, section{Version: version, Date: date, Entries: currentEntries})
	}

	for i := len(tags) - 1; i >= 0; i-- {
		rangeSpec := tags[i].Name
		if i > 0 {
			rangeSpec = tags[i-1].Name + ".." + tags[i].Name
		}
		entries, err := entriesForRange(rangeSpec, opts.StrictOnly)
		if err != nil {
			return nil, err
		}
		result = append(result, section{
			Version: gitutil.VersionFromTag(tags[i].Name),
			Date:    tags[i].Date,
			Entries: entries,
		})
	}

	if len(result) == 0 {
		result = append(result, section{Version: "Unreleased"})
	}
	return result, nil
}

func entriesForRange(rangeSpec string, strictOnly bool) ([]entry, error) {
	commits, err := gitutil.Log(rangeSpec)
	if err != nil {
		return nil, err
	}

	entries := make([]entry, 0, len(commits))
	for _, gitCommit := range commits {
		parsed, ok := commit.Parse(gitCommit.Message)
		if strictOnly && !ok {
			continue
		}
		if ok && parsed.Type == "chore" && parsed.Scope == "release" {
			continue
		}

		item := entry{Hash: gitCommit.ShortHash(), Conventional: ok}
		if ok {
			item.Type = parsed.Type
			item.Scope = parsed.Scope
			item.Description = parsed.Description
			item.Footer = parsed.Footer
		} else {
			item.Type = "other"
			item.Description = firstLine(gitCommit.Message)
		}
		entries = append(entries, item)
	}
	return entries, nil
}

func render(sections []section) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# Changelog")
	fmt.Fprintln(&b)

	for _, s := range sections {
		if s.Date != "" {
			fmt.Fprintf(&b, "## %s - %s\n\n", s.Version, s.Date)
		} else {
			fmt.Fprintf(&b, "## %s\n\n", s.Version)
		}

		if len(s.Entries) == 0 {
			fmt.Fprintln(&b, "_No matching commits found._")
			fmt.Fprintln(&b)
			continue
		}

		byType := map[string][]entry{}
		for _, item := range s.Entries {
			byType[item.Type] = append(byType[item.Type], item)
		}

		for _, group := range sortedTypes(byType) {
			fmt.Fprintf(&b, "### %s\n\n", commit.TitleFor(group))
			for _, item := range byType[group] {
				scope := ""
				if item.Scope != "" {
					scope = fmt.Sprintf("**%s:** ", item.Scope)
				}
				footer := ""
				if item.Footer != "" {
					footer = fmt.Sprintf(" _(%s)_", item.Footer)
				}
				fmt.Fprintf(&b, "- %s%s (`%s`)%s\n", scope, item.Description, item.Hash, footer)
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func sortedTypes(groups map[string][]entry) []string {
	preferred := []string{"feat", "fix", "perf", "refactor", "docs", "test", "build", "ci", "chore", "style", "revert", "other"}
	seen := map[string]bool{}
	var ordered []string
	for _, key := range preferred {
		if _, ok := groups[key]; ok {
			ordered = append(ordered, key)
			seen[key] = true
		}
	}
	var rest []string
	for key := range groups {
		if !seen[key] {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)
	return append(ordered, rest...)
}

func firstLine(message string) string {
	line, _, _ := strings.Cut(message, "\n")
	return strings.TrimSpace(line)
}
