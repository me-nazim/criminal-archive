// Package cases owns the Case resource: the central object of the
// archive. It manages the submission state machine, person + timeline +
// news linkage, and atomic case-number generation.
package cases

// allowedTransitions maps current_status -> set of valid next states.
// A case is `published` only when it has been both verified by a
// moderator and approved for publication by an admin; we model that by
// requiring `approved` (admin) and a successful verification row before
// `published`.
var allowedTransitions = map[string]map[string]bool{
	"draft":           {"pending_review": true, "archived": true},
	"pending_review":  {"in_verification": true, "rejected": true, "archived": true, "draft": true},
	"in_verification": {"approved": true, "rejected": true, "pending_review": true},
	"approved":        {"published": true, "rejected": true, "archived": true},
	"published":       {"archived": true},
	"rejected":        {"draft": true, "archived": true},
	"archived":        {"draft": true, "published": true},
}

// CanTransition reports whether moving from `from` to `to` is allowed.
func CanTransition(from, to string) bool {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

// Statuses returns every legal status value (handy for filter
// validation).
func Statuses() []string {
	return []string{
		"draft", "pending_review", "in_verification",
		"approved", "published", "rejected", "archived",
	}
}
