// Package workflowrun starts a finance workflow and waits for it to finish,
// for every transport that calls the workflows: the MCP server and the web
// gateway.
package workflowrun

import (
	"context"

	"github.com/google/uuid"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// TaskQueueName is the Temporal task queue the worker registers every finance
// workflow on, and the one every workflow is started on.
const TaskQueueName = "finance"

// Option changes the options a workflow is started with, on top of the
// finance task queue. Setting WorkflowExecutionTimeout bounds how long the
// caller waits: a workflow whose task panics is otherwise retried, and the
// caller blocked, until a fixed worker completes it.
type Option func(*client.StartWorkflowOptions)

func startOptions(opts []Option) client.StartWorkflowOptions {
	options := client.StartWorkflowOptions{TaskQueue: TaskQueueName}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

// Joining starts the workflow under id, or joins the run already under it, so
// two requests for the same deletion wait on one run (ADR 0012). It clears
// the execution timeout, which would cut a long deletion in half, so the
// caller bounds its wait on its own.
func Joining(id string) Option {
	return func(o *client.StartWorkflowOptions) {
		o.ID = id
		o.WorkflowIDConflictPolicy = enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING
		o.WorkflowExecutionTimeout = 0
	}
}

// Start starts the workflow wf with in, as the user ctx acts as, and returns
// its run without waiting for it. The run's ID is the one opts set, or a
// random one, behind the user's prefix: added here, after every option, so
// no caller can start a run under another user's ID (ADR 0013).
func Start(ctx context.Context, c client.Client, wf any, in any, opts ...Option) (client.WorkflowRun, error) {
	uid, ok := tenancy.UserFrom(ctx)
	if !ok {
		return nil, internalerrors.ErrMissingRequiredParameter{Parameters: []string{"user"}}
	}
	options := startOptions(opts)
	if options.ID == "" {
		options.ID = uuid.NewString()
	}
	options.ID = tenancy.WorkflowIDPrefix(uid) + options.ID
	return c.ExecuteWorkflow(ctx, options, wf, in)
}

// Run starts the workflow wf with in, waits for it to finish, and returns its
// result decoded into an O.
func Run[O any](ctx context.Context, c client.Client, wf any, in any, opts ...Option) (*O, error) {
	run, err := Start(ctx, c, wf, in, opts...)
	if err != nil {
		return nil, err
	}

	result := new(O)
	if err := run.Get(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

// RunWithoutResult starts the workflow wf with in and waits for it to finish,
// for a workflow that returns only an error.
func RunWithoutResult(ctx context.Context, c client.Client, wf any, in any, opts ...Option) error {
	run, err := Start(ctx, c, wf, in, opts...)
	if err != nil {
		return err
	}
	return run.Get(ctx, nil)
}
