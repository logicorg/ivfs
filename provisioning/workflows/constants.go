// Package workflows contains the provisioning Temporal workflows.
package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TaskQueueName is the task queue of the provisioning worker. It is not the
// finance one, so a provisioning activity, which takes the uid it acts on as
// an argument, never runs on a worker that trusts the uid in the context.
const TaskQueueName = "provisioning"

// DefaultActivityOptions retry what a database hiccup can fail, and nothing
// a retry would only fail again: an invalid identity or an unknown uid.
var DefaultActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    10 * time.Second,
		MaximumAttempts:    5,
		NonRetryableErrorTypes: []string{
			"ValidationError",
			"ErrMissingRequiredParameter",
			"ErrNotFound",
		},
	},
}
