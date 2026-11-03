// Package workflows contains the finance related Temporal workflows
package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var (
	// DefaultActivityOptions are the default options for activity execution.
	DefaultActivityOptions = workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    5,
			NonRetryableErrorTypes: []string{
				"ValidationError",
				"ErrInvalidParameter",
				"ErrMissingRequiredParameter",
				"ErrNotFound",
				"ErrResourceClosed",
				"ErrResourceConflict",
			},
		},
	}
)

// Opts are the options for activity options
type Opts struct {
	RetryableNotFound bool
}

// GetDefaultActivityOptions returns the default activity options
func GetDefaultActivityOptions(opts ...Opts) workflow.ActivityOptions {
	if len(opts) == 0 {
		return DefaultActivityOptions
	}

	ao := DefaultActivityOptions
	if !opts[0].RetryableNotFound {
		ao.RetryPolicy.NonRetryableErrorTypes = append(ao.RetryPolicy.NonRetryableErrorTypes, "ErrNotFound")
	}
	return ao
}
