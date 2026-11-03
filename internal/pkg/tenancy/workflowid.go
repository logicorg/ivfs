package tenancy

import "github.com/google/uuid"

// WorkflowIDPrefix is what the ID of every workflow a user starts begins
// with. Two users deleting their account 7 then run under two IDs, rather
// than one joining the other's run (ADR 0012), and the worker can tell from a
// run's root ID whose it is (ADR 0013).
func WorkflowIDPrefix(uid uuid.UUID) string {
	return uid.String() + "/"
}
