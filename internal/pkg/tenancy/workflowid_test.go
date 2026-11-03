package tenancy

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkflowIDPrefix checks that the prefix is the user's id and a
// separator, which keeps the rest of the ID readable on its own.
func TestWorkflowIDPrefix(t *testing.T) {
	tests := map[string]struct {
		uid  uuid.UUID
		want string
	}{
		"a user": {
			uid:  uuid.MustParse("0192f3a0-0000-7000-8000-000000000001"),
			want: "0192f3a0-0000-7000-8000-000000000001/",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, WorkflowIDPrefix(tc.uid))
		})
	}
}
