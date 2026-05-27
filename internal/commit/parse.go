package commit

import (
	"regexp"
	"strings"
)

var conventionalPattern = regexp.MustCompile(`^([a-z]+)(\([^)]+\))?(!)?: (.+)$`)
var footerTokenPattern = regexp.MustCompile(`^([A-Za-z-]+|BREAKING CHANGE): .+$`)
var issueReferencePattern = regexp.MustCompile(`(?i)\b(closes?|fix(es|ed)?|resolves?|refs?)\s+#\d+`)

// Parsed is a structured Conventional Commit message.
type Parsed struct {
	Type        string
	Scope       string
	Description string
	Breaking    bool
	Footer      string
	Raw         string
}

// Parse returns a parsed commit when the first line follows Conventional Commits.
func Parse(raw string) (Parsed, bool) {
	lines := splitLines(raw)
	if len(lines) == 0 {
		return Parsed{}, false
	}

	matches := conventionalPattern.FindStringSubmatch(lines[0])
	if matches == nil {
		return Parsed{Raw: raw}, false
	}

	scope := matches[2]
	if len(scope) >= 2 {
		scope = scope[1 : len(scope)-1]
	}

	parsed := Parsed{
		Type:        matches[1],
		Scope:       scope,
		Description: matches[4],
		Breaking:    matches[3] == "!" || hasBreakingFooter(lines),
		Footer:      footerFromLines(lines),
		Raw:         raw,
	}
	return parsed, true
}

func splitLines(raw string) []string {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), "\r\n", "\n")
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "\n")
}

func hasBreakingFooter(lines []string) bool {
	for _, line := range lines[1:] {
		if strings.HasPrefix(strings.TrimSpace(line), "BREAKING CHANGE:") {
			return true
		}
	}
	return false
}

func footerFromLines(lines []string) string {
	var footers []string
	for i := len(lines) - 1; i >= 1; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			if len(footers) > 0 {
				break
			}
			continue
		}
		if !isFooterLine(line) {
			break
		}
		footers = append([]string{line}, footers...)
	}
	return strings.Join(footers, "; ")
}

func isFooterLine(line string) bool {
	return footerTokenPattern.MatchString(line) || issueReferencePattern.MatchString(line)
}
