package version

import "testing"

func TestParseSemver(t *testing.T) {
	got, err := Parse("1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "1.2.3" {
		t.Fatalf("got %s", got)
	}
}

func TestBump(t *testing.T) {
	base := Semver{Major: 1, Minor: 2, Patch: 3}
	tests := map[string]string{
		"major": "2.0.0",
		"minor": "1.3.0",
		"patch": "1.2.4",
	}
	for kind, want := range tests {
		if got := base.Bump(kind).String(); got != want {
			t.Fatalf("%s bump = %s, want %s", kind, got, want)
		}
	}
}
