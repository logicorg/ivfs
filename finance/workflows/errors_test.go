package workflows

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

// TestHasErrorType checks that the type is found anywhere in the chain, as
// when a child workflow wraps an activity's structured error: the first
// ApplicationError is then the child's wrapper, and the structured one is
// further down. FindErrorType returns that structured error, whose message
// is the activity's own.
func TestHasErrorType(t *testing.T) {
	notFound := temporal.NewApplicationError("resource Institution not found", "ErrNotFound")
	childWrapper := temporal.NewApplicationErrorWithCause("retrieve institution: resource Institution not found", "wrapError", notFound)

	tests := map[string]struct {
		err  error
		want bool
	}{
		"the structured error itself":                    {err: notFound, want: true},
		"under a child workflow's wrapper":               {err: childWrapper, want: true},
		"under the wrapper, wrapped again by the facade": {err: fmt.Errorf("open account: %w", childWrapper), want: true},
		"another type":  {err: temporal.NewApplicationError("closed", "ErrResourceClosed"), want: false},
		"a plain error": {err: errors.New("resource Institution not found"), want: false},
		"no error":      {err: nil, want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, HasErrorType(tc.err, "ErrNotFound"))

			found, ok := FindErrorType(tc.err, "ErrNotFound")
			require.Equal(t, tc.want, ok)
			if tc.want {
				require.Same(t, notFound, found)
			} else {
				require.Nil(t, found)
			}
		})
	}
}
