package tenancy

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Interceptor refuses, on the finance worker, a workflow or an activity that
// does not run as a user, and a workflow whose chain was not started behind
// its user's prefix (ADR 0013). The propagator has already read the header
// when it runs, so a run carrying a malformed user never gets this far.
//
// It catches what the gateway never sends but anyone else reaching Temporal
// could, such as a `temporal workflow start` with no header, or with one
// user's header on another's workflow ID. The provisioning worker must not
// use it: its workflows run before the user has a session.
func Interceptor() interceptor.WorkerInterceptor {
	return &workerInterceptor{}
}

type workerInterceptor struct {
	interceptor.WorkerInterceptorBase
}

func (*workerInterceptor) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	i := &activityInbound{}
	i.Next = next
	return i
}

func (*workerInterceptor) InterceptWorkflow(
	_ workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	i := &workflowInbound{}
	i.Next = next
	return i
}

type activityInbound struct {
	interceptor.ActivityInboundInterceptorBase
}

// An activity is checked for a user only: it got it from the workflow that
// scheduled it, which was checked against its root ID.
func (i *activityInbound) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (any, error) {
	if _, ok := UserFrom(ctx); !ok {
		return nil, refuse(internalerrors.ErrMissingRequiredParameter{Parameters: []string{"user"}})
	}
	return i.Next.ExecuteActivity(ctx, in)
}

type workflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
}

// The root, not the workflow's own ID, is what is checked: children keep
// the SDK's IDs, but every chain starts at a run the gateway named.
func (i *workflowInbound) ExecuteWorkflow(
	ctx workflow.Context,
	in *interceptor.ExecuteWorkflowInput,
) (any, error) {
	uid, ok := WorkflowUser(ctx)
	if !ok {
		return nil, refuse(internalerrors.ErrMissingRequiredParameter{Parameters: []string{"user"}})
	}
	info := workflow.GetInfo(ctx)
	root := info.WorkflowExecution.ID
	if info.RootWorkflowExecution != nil {
		root = info.RootWorkflowExecution.ID
	}
	if !strings.HasPrefix(root, WorkflowIDPrefix(uid)) {
		return nil, refuse(internalerrors.ErrInvalidParameter{Description: fmt.Sprintf(
			"workflow %s runs in a chain started as %s, which user %s did not start; "+
				"start workflows through the gateway", info.WorkflowExecution.ID, root, uid)})
	}
	return i.Next.ExecuteWorkflow(ctx, in)
}

// refuse fails the run for good: no retry policy, the caller's or the
// finance defaults, can turn a run as no one into a run as someone.
func refuse(err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), reflect.TypeOf(err).Name(), err)
}
