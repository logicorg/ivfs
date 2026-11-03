package gateway_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/logicorg/ivfs/web/api/gateway"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TestWorkflowError checks that each structured error type becomes its status
// with the activity's own message, found anywhere in the error's chain, and
// that any other error becomes a 500 that does not repeat it.
func TestWorkflowError(t *testing.T) {
	tests := map[string]struct {
		err        error
		wantStatus int
		wantDetail string
	}{
		"a validation error": {
			err:        apitest.ChildWorkflowError("ValidationError", "code: invalid value"),
			wantStatus: http.StatusUnprocessableEntity, wantDetail: "code: invalid value",
		},
		"an invalid parameter": {
			err:        apitest.ChildWorkflowError("ErrInvalidParameter", "amount must be positive"),
			wantStatus: http.StatusUnprocessableEntity, wantDetail: "amount must be positive",
		},
		"a missing parameter": {
			err:        apitest.ChildWorkflowError("ErrMissingRequiredParameter", "missing name"),
			wantStatus: http.StatusUnprocessableEntity, wantDetail: "missing name",
		},
		"a resource that does not exist": {
			err:        apitest.ChildWorkflowError("ErrNotFound", "resource Institution not found"),
			wantStatus: http.StatusNotFound, wantDetail: "resource Institution not found",
		},
		"a conflict": {
			err:        apitest.ChildWorkflowError("ErrResourceConflict", "name already used"),
			wantStatus: http.StatusConflict, wantDetail: "name already used",
		},
		"a closed resource": {
			err:        apitest.ChildWorkflowError("ErrResourceClosed", "financial account 3 is closed"),
			wantStatus: http.StatusConflict, wantDetail: "financial account 3 is closed",
		},
		"the structured error, not wrapped": {
			err:        temporal.NewApplicationError("resource Currency not found", "ErrNotFound"),
			wantStatus: http.StatusNotFound, wantDetail: "resource Currency not found",
		},
		"an error of another type": {
			err:        apitest.ChildWorkflowError("PanicError", "nil pointer dereference"),
			wantStatus: http.StatusInternalServerError, wantDetail: "the workflow failed; the gateway's log has the details",
		},
		"a timed-out workflow": {
			err:        temporal.NewTimeoutError(0, nil),
			wantStatus: http.StatusInternalServerError, wantDetail: "the workflow failed; the gateway's log has the details",
		},
		"a plain error": {
			err:        errors.New("connection refused"),
			wantStatus: http.StatusInternalServerError, wantDetail: "the workflow failed; the gateway's log has the details",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var problem *huma.ErrorModel
			require.ErrorAs(t, gateway.WorkflowError(tc.err), &problem)
			require.Equal(t, tc.wantStatus, problem.Status)
			require.Equal(t, tc.wantDetail, problem.Detail)
		})
	}
}

func echoWorkflow(_ workflow.Context, in string) (*string, error) { return &in, nil }

// TestRunWorkflow checks that a workflow is started with the default timeout,
// unless the endpoint sets its own, and that its result or its mapped error
// is returned.
func TestRunWorkflow(t *testing.T) {
	tests := map[string]struct {
		opts        []workflowrun.Option
		runErr      error
		wantTimeout time.Duration
		wantStatus  int
	}{
		"the default timeout": {
			wantTimeout: gateway.WorkflowTimeout,
		},
		"the endpoint's own timeout": {
			opts:        []workflowrun.Option{func(o *client.StartWorkflowOptions) { o.WorkflowExecutionTimeout = time.Minute }},
			wantTimeout: time.Minute,
		},
		"a failed workflow": {
			runErr:      temporal.NewApplicationError("resource Currency not found", "ErrNotFound"),
			wantTimeout: gateway.WorkflowTimeout,
			wantStatus:  http.StatusNotFound,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			run := &mocks.WorkflowRun{}
			run.On("Get", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) { *args.Get(1).(*string) = "out" }).
				Return(tc.runErr)
			c := &mocks.Client{}
			c.On("ExecuteWorkflow", mock.Anything, mock.MatchedBy(func(o client.StartWorkflowOptions) bool {
				return o.TaskQueue == workflowrun.TaskQueueName && o.WorkflowExecutionTimeout == tc.wantTimeout
			}), mock.Anything, "in").Return(run, nil)

			result, err := gateway.RunWorkflow[string](tenancy.WithUser(t.Context(), apitest.User), c, echoWorkflow, "in", tc.opts...)
			c.AssertExpectations(t)
			if tc.wantStatus != 0 {
				var problem *huma.ErrorModel
				require.ErrorAs(t, err, &problem)
				require.Equal(t, tc.wantStatus, problem.Status)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "out", *result)
		})
	}
}
