package gateway

import (
	"context"
	"time"

	"go.temporal.io/sdk/client"
)

// RunDeletionWithin is RunDeletion with a wait short enough for a test.
func RunDeletionWithin[O any](ctx context.Context, c client.Client, wait time.Duration, workflowID string, wf, in any) (*O, bool, error) {
	return runDeletion[O](ctx, c, wait, workflowID, wf, in)
}
