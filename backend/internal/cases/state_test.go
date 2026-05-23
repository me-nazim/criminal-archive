package cases

import "testing"

func TestCanTransition(t *testing.T) {
	t.Parallel()
	cases := []struct {
		from, to string
		want     bool
	}{
		// Allowed
		{"draft", "pending_review", true},
		{"pending_review", "in_verification", true},
		{"in_verification", "approved", true},
		{"in_verification", "rejected", true},
		{"approved", "published", true},
		{"published", "archived", true},
		{"archived", "draft", true}, // resurrect from archive
		{"archived", "published", true},

		// Disallowed
		{"draft", "published", false}, // skipping review
		{"draft", "approved", false},
		{"published", "approved", false},
		{"rejected", "approved", false},
		{"unknown", "draft", false},
		{"draft", "draft", false},
	}
	for _, tc := range cases {
		got := CanTransition(tc.from, tc.to)
		if got != tc.want {
			t.Errorf("CanTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestStatusesContainsAllUsedStates(t *testing.T) {
	t.Parallel()
	all := Statuses()
	known := map[string]bool{}
	for _, s := range all {
		known[s] = true
	}
	for _, s := range []string{"draft", "pending_review", "in_verification", "approved", "published", "rejected", "archived"} {
		if !known[s] {
			t.Errorf("Statuses() missing %q", s)
		}
	}
}
