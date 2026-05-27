package commit

import "testing"

func TestParseConventionalCommit(t *testing.T) {
	parsed, ok := Parse("feat(cli)!: add commit builder\n\nCloses #42")
	if !ok {
		t.Fatal("expected conventional commit")
	}
	if parsed.Type != "feat" || parsed.Scope != "cli" || !parsed.Breaking {
		t.Fatalf("unexpected parse result: %#v", parsed)
	}
	if parsed.Description != "add commit builder" || parsed.Footer != "Closes #42" {
		t.Fatalf("unexpected text fields: %#v", parsed)
	}
}

func TestParseRejectsNonConventionalCommit(t *testing.T) {
	_, ok := Parse("updated stuff")
	if ok {
		t.Fatal("expected non-conventional commit to be rejected")
	}
}

func TestParseBreakingFooter(t *testing.T) {
	parsed, ok := Parse("feat(api): rename endpoint\n\nBREAKING CHANGE: clients must use /v2")
	if !ok {
		t.Fatal("expected conventional commit")
	}
	if !parsed.Breaking {
		t.Fatal("expected breaking change footer to mark commit as breaking")
	}
	if parsed.Footer != "BREAKING CHANGE: clients must use /v2" {
		t.Fatalf("unexpected footer %q", parsed.Footer)
	}
}
