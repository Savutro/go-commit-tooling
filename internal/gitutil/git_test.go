package gitutil

import "testing"

func TestParseStatusLine(t *testing.T) {
	tests := map[string]StatusEntry{
		"M  README.md": {
			Path:           "README.md",
			DisplayPath:    "README.md",
			IndexStatus:    'M',
			WorktreeStatus: ' ',
		},
		"?? internal/stage/stage.go": {
			Path:           "internal/stage/stage.go",
			DisplayPath:    "internal/stage/stage.go",
			IndexStatus:    '?',
			WorktreeStatus: '?',
		},
		"R  old.go -> new.go": {
			Path:           "new.go",
			DisplayPath:    "old.go -> new.go",
			IndexStatus:    'R',
			WorktreeStatus: ' ',
		},
	}

	for line, want := range tests {
		got, ok := parseStatusLine(line)
		if !ok {
			t.Fatalf("parseStatusLine(%q) failed", line)
		}
		if got != want {
			t.Fatalf("parseStatusLine(%q) = %#v, want %#v", line, got, want)
		}
	}
}

func TestStatusEntryStaged(t *testing.T) {
	if !(StatusEntry{IndexStatus: 'M'}).Staged() {
		t.Fatal("expected modified index status to be staged")
	}
	if (StatusEntry{IndexStatus: '?'}).Staged() {
		t.Fatal("expected untracked status to be unstaged")
	}
}
