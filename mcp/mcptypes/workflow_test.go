package mcptypes

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/workflow"
)

func echoWorkflow(_ workflow.Context, in string) (string, error) { return in, nil }

// TestWorkflowAdapters checks that both adapters run their workflow with the
// tool's arguments and start options, and that WorkflowWithResult returns the
// workflow's result. How the options are applied, and the user required, is
// workflowrun's, and tested there.
func TestWorkflowAdapters(t *testing.T) {
	timeout := func(o *client.StartWorkflowOptions) { o.WorkflowExecutionTimeout = time.Minute }
	wantOptions := mock.MatchedBy(func(o client.StartWorkflowOptions) bool {
		return o.TaskQueue == workflowrun.TaskQueueName && o.WorkflowExecutionTimeout == time.Minute
	})
	// The MCP server has no user to act as, so it cannot run a workflow
	// (ADR 0013); the adapters are tested as if it had one.
	ctx := tenancy.WithUser(t.Context(), uuid.MustParse("0192f3a0-0000-7000-8000-000000000001"))

	tests := map[string]struct {
		// call runs the adapter's handler against c and returns its result.
		call func(t *testing.T, c client.Client) any
		want any
	}{
		"WorkflowWithResult": {
			call: func(t *testing.T, c client.Client) any {
				handler := WorkflowWithResult[func(workflow.Context, string) (string, error), string, string](c, echoWorkflow, timeout)
				_, result, err := handler(ctx, nil, "in")
				require.NoError(t, err)
				return *result
			},
			want: "out",
		},
		"WorkflowEmptyResult": {
			call: func(t *testing.T, c client.Client) any {
				handler := WorkflowEmptyResult[string](c, echoWorkflow, timeout)
				_, result, err := handler(ctx, nil, "in")
				require.NoError(t, err)
				return result
			},
			want: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			run := &mocks.WorkflowRun{}
			run.On("Get", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					if result, ok := args.Get(1).(*string); ok {
						*result = "out"
					}
				}).
				Return(nil)
			c := &mocks.Client{}
			c.On("ExecuteWorkflow", mock.Anything, wantOptions, mock.Anything, "in").Return(run, nil)

			require.Equal(t, tc.want, tc.call(t, c))
			c.AssertExpectations(t)
		})
	}
}
