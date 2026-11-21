package gateway

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/logicorg/ivfs/finance/workflows"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"go.temporal.io/sdk/client"
)

// WorkflowTimeout bounds how long an endpoint waits for its workflow. A
// workflow whose task panics is retried until a fixed worker completes it,
// and the request would otherwise wait for as long.
const WorkflowTimeout = 30 * time.Second

// errorStatuses are the statuses a workflow's structured errors are answered
// with, in the order they are looked for in the error's chain.
var errorStatuses = []struct {
	errType string
	status  int
}{
	{"ValidationError", http.StatusUnprocessableEntity},
	{"ErrInvalidParameter", http.StatusUnprocessableEntity},
	{"ErrMissingRequiredParameter", http.StatusUnprocessableEntity},
	{"ErrNotFound", http.StatusNotFound},
	{"ErrResourceConflict", http.StatusConflict},
	{"ErrResourceClosed", http.StatusConflict},
}

// RunWorkflow runs the workflow wf with in, waits at most WorkflowTimeout, or
// as long as opts set, and returns its result, or its error as the problem
// the API answers with.
func RunWorkflow[O any](ctx context.Context, c client.Client, wf any, in any, opts ...workflowrun.Option) (*O, error) {
	result, err := workflowrun.Run[O](ctx, c, wf, in, withTimeout(opts)...)
	if err != nil {
		return nil, WorkflowError(ctx, err)
	}
	return result, nil
}

// RunWorkflowWithoutResult is RunWorkflow for a workflow that returns only
// an error.
func RunWorkflowWithoutResult(ctx context.Context, c client.Client, wf any, in any, opts ...workflowrun.Option) error {
	if err := workflowrun.RunWithoutResult(ctx, c, wf, in, withTimeout(opts)...); err != nil {
		return WorkflowError(ctx, err)
	}
	return nil
}

// withTimeout puts WorkflowTimeout before opts, so an endpoint's own timeout
// wins.
func withTimeout(opts []workflowrun.Option) []workflowrun.Option {
	return append([]workflowrun.Option{func(o *client.StartWorkflowOptions) {
		o.WorkflowExecutionTimeout = WorkflowTimeout
	}}, opts...)
}

// WorkflowError turns a workflow's error into a problem: a structured error
// anywhere in its chain, such as one a facade's child workflow wrapped,
// becomes its status with the error's own message, and anything else is
// logged and answered with a 500 that does not repeat it.
//
// The one exception is the user ctx acts as missing from the registry: that
// is the session's fault, not the request's, so it is a 401, which signs the
// browser out rather than showing a 404 on every page.
func WorkflowError(ctx context.Context, err error) error {
	if uid, ok := tenancy.UserFrom(ctx); ok {
		if found, ok := workflows.FindErrorType(err, "ErrNotFound"); ok && found.Message() == tenancy.UnknownUser(uid).Error() {
			return huma.Error401Unauthorized("this session's user no longer exists; sign in again")
		}
	}
	for _, s := range errorStatuses {
		if found, ok := workflows.FindErrorType(err, s.errType); ok {
			return huma.NewError(s.status, found.Message())
		}
	}
	log.Printf("workflow failed: %v", err)
	return huma.Error500InternalServerError("the workflow failed; the gateway's log has the details")
}
