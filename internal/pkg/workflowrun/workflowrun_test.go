package workflowrun

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/workflow"
)

func echoWorkflow(_ workflow.Context, in string) (string, error) { return in, nil }

// runners calls Run and RunWithoutResult the same way, so one table covers
// both. RunWithoutResult has no result, so it returns "".
var runners = map[string]func(ctx context.Context, c client.Client, opts ...Option) (string, error){
	"Run": func(ctx context.Context, c client.Client, opts ...Option) (string, error) {
		result, err := Run[string](ctx, c, echoWorkflow, "in", opts...)
		if err != nil {
			return "", err
		}
		return *result, nil
	},
	"RunWithoutResult": func(ctx context.Context, c client.Client, opts ...Option) (string, error) {
		return "", RunWithoutResult(ctx, c, echoWorkflow, "in", opts...)
	},
}

// TestStartOptions checks that a workflow is started on the finance task
// queue, with the caller's options applied in order on top, such as a timeout
// that bounds how long the caller waits for a workflow that cannot finish.
func TestStartOptions(t *testing.T) {
	tests := map[string]struct {
		opts []Option
		want client.StartWorkflowOptions
	}{
		"no options": {
			want: client.StartWorkflowOptions{TaskQueue: TaskQueueName},
		},
		"an execution timeout": {
			opts: []Option{
				func(o *client.StartWorkflowOptions) { o.WorkflowExecutionTimeout = time.Minute },
			},
			want: client.StartWorkflowOptions{TaskQueue: TaskQueueName, WorkflowExecutionTimeout: time.Minute},
		},
		"options applied in order": {
			opts: []Option{
				func(o *client.StartWorkflowOptions) { o.WorkflowExecutionTimeout = time.Minute },
				func(o *client.StartWorkflowOptions) { o.WorkflowExecutionTimeout = 2 * time.Minute },
			},
			want: client.StartWorkflowOptions{TaskQueue: TaskQueueName, WorkflowExecutionTimeout: 2 * time.Minute},
		},
		"joining a run under an id": {
			opts: []Option{
				func(o *client.StartWorkflowOptions) { o.WorkflowExecutionTimeout = time.Minute },
				Joining("delete-institution-7"),
			},
			want: client.StartWorkflowOptions{
				TaskQueue: TaskQueueName, ID: "delete-institution-7",
				WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, startOptions(tc.opts))
		})
	}
}

// TestStart checks that both runners start a workflow only as a user, under
// an ID behind that user's prefix, whatever ID the options set.
func TestStart(t *testing.T) {
	uid := uuid.MustParse("0192f3a0-0000-7000-8000-000000000001")
	other := uuid.MustParse("0192f3a0-0000-7000-8000-000000000002")

	tests := map[string]struct {
		ctx  context.Context
		opts []Option
		// wantID is the ID after the prefix; "*" is any UUID.
		wantID  string
		wantErr string
	}{
		"no user": {
			ctx:     t.Context(),
			wantErr: "missing required parameters: user",
		},
		"no ID": {
			ctx:    tenancy.WithUser(t.Context(), uid),
			wantID: "*",
		},
		"an ID": {
			ctx:    tenancy.WithUser(t.Context(), uid),
			opts:   []Option{Joining("delete-institution-7")},
			wantID: "delete-institution-7",
		},
		"an ID behind another user's prefix": {
			ctx:    tenancy.WithUser(t.Context(), uid),
			opts:   []Option{Joining(tenancy.WorkflowIDPrefix(other) + "delete-institution-7")},
			wantID: tenancy.WorkflowIDPrefix(other) + "delete-institution-7",
		},
	}

	for runnerName, runner := range runners {
		for name, tc := range tests {
			t.Run(runnerName+"/"+name, func(t *testing.T) {
				run := &mocks.WorkflowRun{}
				run.On("Get", mock.Anything, mock.Anything).Return(nil)
				c := &mocks.Client{}
				var gotID string
				c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, "in").
					Run(func(args mock.Arguments) { gotID = args.Get(1).(client.StartWorkflowOptions).ID }).
					Return(run, nil).Maybe()

				_, err := runner(tc.ctx, c, tc.opts...)
				if tc.wantErr != "" {
					require.ErrorContains(t, err, tc.wantErr)
					c.AssertNotCalled(t, "ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
					return
				}
				require.NoError(t, err)
				rest, ok := strings.CutPrefix(gotID, tenancy.WorkflowIDPrefix(uid))
				require.True(t, ok, "ID %q is not behind the user's prefix", gotID)
				if tc.wantID == "*" {
					require.NoError(t, uuid.Validate(rest))
					return
				}
				require.Equal(t, tc.wantID, rest)
			})
		}
	}
}

// TestRun checks that Run returns the workflow's result, and that both
// runners return the error of a workflow that could not start or failed.
func TestRun(t *testing.T) {
	tests := map[string]struct {
		startErr, runErr error
		// runners names the runners the case applies to.
		runners   []string
		want      string
		wantError string
	}{
		"the workflow's result": {
			runners: []string{"Run"},
			want:    "out",
		},
		"a workflow that could not start": {
			startErr:  errors.New("temporal unavailable"),
			runners:   []string{"Run", "RunWithoutResult"},
			wantError: "temporal unavailable",
		},
		"a workflow that failed": {
			runErr:    errors.New("workflow failed"),
			runners:   []string{"Run", "RunWithoutResult"},
			wantError: "workflow failed",
		},
	}

	for name, tc := range tests {
		for _, runnerName := range tc.runners {
			t.Run(runnerName+"/"+name, func(t *testing.T) {
				run := &mocks.WorkflowRun{}
				run.On("Get", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						if result, ok := args.Get(1).(*string); ok {
							*result = "out"
						}
					}).
					Return(tc.runErr)
				c := &mocks.Client{}
				c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, "in").Return(run, tc.startErr)

				got, err := runners[runnerName](tenancy.WithUser(t.Context(), uuid.New()), c)
				if tc.wantError != "" {
					require.ErrorContains(t, err, tc.wantError)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			})
		}
	}
}
