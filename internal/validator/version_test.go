package validator

import (
	"testing"
)

func TestIsBreakingBump(t *testing.T) {
	cases := []struct {
		intent  string
		impl    string
		want    bool
	}{
		{"2.0.0", "1.0.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.1.0", "1.0.0", false},
		{"1.0.1", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
	}
	for _, c := range cases {
		got, err := IsBreakingBump(c.intent, c.impl)
		if err != nil {
			t.Errorf("IsBreakingBump(%q, %q): unexpected error: %v", c.intent, c.impl, err)
			continue
		}
		if got != c.want {
			t.Errorf("IsBreakingBump(%q, %q) = %v, want %v", c.intent, c.impl, got, c.want)
		}
	}
}

func TestBumpKind(t *testing.T) {
	cases := []struct {
		from string
		to   string
		want string
	}{
		{"1.0.0", "2.0.0", "major"},
		{"1.0.0", "1.1.0", "minor"},
		{"1.0.0", "1.0.1", "patch"},
		{"1.0.0", "1.0.0", "unknown"},
	}
	for _, c := range cases {
		got := BumpKind(c.from, c.to)
		if got != c.want {
			t.Errorf("BumpKind(%q, %q) = %q, want %q", c.from, c.to, got, c.want)
		}
	}
}
