// Package gitutil contains the small Git command wrappers used by gct.
package gitutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// Commit is a single Git commit with its full message body.
type Commit struct {
	Hash    string
	Message string
}

// ShortHash returns the seven-character hash commonly used in changelogs.
func (c Commit) ShortHash() string {
	if len(c.Hash) <= 7 {
		return c.Hash
	}
	return c.Hash[:7]
}

// Tag is a version tag merged into HEAD.
type Tag struct {
	Name string
	Date string
}

// StatusEntry is one changed path from git status porcelain output.
type StatusEntry struct {
	Path           string
	DisplayPath    string
	IndexStatus    byte
	WorktreeStatus byte
}

// Staged reports whether the entry currently has staged changes.
func (e StatusEntry) Staged() bool {
	return e.IndexStatus != ' ' && e.IndexStatus != '?' && e.IndexStatus != '!'
}

// Label formats a status entry for the staging picker.
func (e StatusEntry) Label() string {
	index := statusName(e.IndexStatus)
	worktree := statusName(e.WorktreeStatus)
	return fmt.Sprintf("%-11s %-11s %s", index, worktree, e.DisplayPath)
}

// Log returns commits for a Git revision range, newest first.
func Log(rangeSpec string) ([]Commit, error) {
	args := []string{"log", "--pretty=format:%H%x1f%B%x1e"}
	if rangeSpec != "" {
		args = append(args, rangeSpec)
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	records := bytes.Split(out, []byte{0x1e})
	commits := make([]Commit, 0, len(records))
	for _, record := range records {
		record = bytes.TrimSpace(record)
		if len(record) == 0 {
			continue
		}
		parts := bytes.SplitN(record, []byte{0x1f}, 2)
		if len(parts) != 2 {
			continue
		}
		commits = append(commits, Commit{
			Hash:    string(parts[0]),
			Message: strings.TrimSpace(string(parts[1])),
		})
	}
	return commits, nil
}

// SemverTags returns merged tags that look like x.y.z or vX.Y.Z, oldest first.
func SemverTags() ([]Tag, error) {
	out, err := exec.Command(
		"git",
		"tag",
		"--merged",
		"HEAD",
		"--sort=creatordate",
		"--format=%(refname:short)%09%(creatordate:short)",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("git tag failed: %w", err)
	}

	var tags []Tag
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		name, date, _ := strings.Cut(line, "\t")
		if IsSemverTag(name) {
			tags = append(tags, Tag{Name: name, Date: date})
		}
	}
	return tags, nil
}

// IsSemverTag reports whether a tag looks like x.y.z or vX.Y.Z.
func IsSemverTag(name string) bool {
	version := strings.TrimPrefix(name, "v")
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// VersionFromTag strips a leading v from a semantic version tag.
func VersionFromTag(name string) string {
	return strings.TrimPrefix(name, "v")
}

// LastSemverTag returns the newest merged semantic version tag, if any.
func LastSemverTag() (Tag, bool, error) {
	tags, err := SemverTags()
	if err != nil {
		return Tag{}, false, err
	}
	if len(tags) == 0 {
		return Tag{}, false, nil
	}
	return tags[len(tags)-1], true, nil
}

// Status returns changed files from the working tree.
func Status() ([]StatusEntry, error) {
	out, err := exec.Command("git", "status", "--porcelain=v1").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	var entries []StatusEntry
	for _, line := range strings.Split(strings.TrimRight(string(out), "\r\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		entry, ok := parseStatusLine(line)
		if ok {
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].DisplayPath < entries[j].DisplayPath
	})
	return entries, nil
}

// TagExists reports whether a tag is already present.
func TagExists(name string) (bool, error) {
	err := exec.Command("git", "rev-parse", "--quiet", "--verify", "refs/tags/"+name).Run()
	if err == nil {
		return true, nil
	}
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("git rev-parse failed: %w", err)
}

// DirtyStatus returns porcelain status output for the working tree.
func DirtyStatus() (string, error) {
	out, err := exec.Command("git", "status", "--porcelain").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Add stages the given paths.
func Add(paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, paths...)
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// RestoreStaged unstages the given paths while leaving worktree changes intact.
func RestoreStaged(paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	args := append([]string{"restore", "--staged", "--"}, paths...)
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("git restore --staged failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// CommitAll creates a commit with a single message.
func CommitAll(message string) error {
	if out, err := exec.Command("git", "commit", "-m", message).CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func parseStatusLine(line string) (StatusEntry, bool) {
	if len(line) < 4 {
		return StatusEntry{}, false
	}
	entry := StatusEntry{
		IndexStatus:    line[0],
		WorktreeStatus: line[1],
	}
	path := strings.TrimSpace(line[3:])
	entry.DisplayPath = path
	entry.Path = path

	if strings.Contains(path, " -> ") {
		parts := strings.Split(path, " -> ")
		entry.Path = parts[len(parts)-1]
	}
	return entry, true
}

func statusName(status byte) string {
	switch status {
	case 'M':
		return "modified"
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case 'U':
		return "conflict"
	case '?':
		return "untracked"
	case '!':
		return "ignored"
	default:
		return "-"
	}
}

// AnnotatedTag creates an annotated tag.
func AnnotatedTag(name, message string) error {
	if out, err := exec.Command("git", "tag", "-a", name, "-m", message).CombinedOutput(); err != nil {
		return fmt.Errorf("git tag failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
