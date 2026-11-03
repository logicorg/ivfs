package tenancy

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// requireRefused checks that err is a refusal of type wantType that no retry
// policy retries, or no error at all when wantType is empty.
func requireRefused(t *testing.T, err error, wantType string) {
	t.Helper()
	if wantType == "" {
		require.NoError(t, err)
		return
	}
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "want an application error, got %v", err)
	require.Equal(t, wantType, appErr.Type())
	require.True(t, appErr.NonRetryable())
}

// TestInterceptorWorkflow checks that a workflow runs only as a user, and
// only in a chain started behind that user's prefix. probeWorkflow runs an
// activity and a child, so a passing case also shows that neither is
// refused: both inherit the user, and the child the root.
func TestInterceptorWorkflow(t *testing.T) {
	uid := uuid.Must(uuid.NewV7())
	other := uuid.Must(uuid.NewV7())
	asUser := header{HeaderKey: payload(t, uid.String())}

	tests := map[string]struct {
		header   header
		id       string
		wantType string
	}{
		"no user": {
			header:   header{},
			id:       WorkflowIDPrefix(uid) + "run",
			wantType: "ErrMissingRequiredParameter",
		},
		"behind the user's prefix": {
			header: asUser,
			id:     WorkflowIDPrefix(uid) + "run",
		},
		"behind another user's prefix": {
			header:   asUser,
			id:       WorkflowIDPrefix(other) + "run",
			wantType: "ErrInvalidParameter",
		},
		"behind no prefix": {
			header:   asUser,
			id:       "run",
			wantType: "ErrInvalidParameter",
		},
		// The slash is part of the prefix, or a uid followed by anything
		// would pass as that user's.
		"the bare uid": {
			header:   asUser,
			id:       uid.String(),
			wantType: "ErrInvalidParameter",
		},
		"the user's prefix further in": {
			header:   asUser,
			id:       "run/" + WorkflowIDPrefix(uid),
			wantType: "ErrInvalidParameter",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var s testsuite.WorkflowTestSuite
			s.SetContextPropagators([]workflow.ContextPropagator{Propagator()})
			env := s.NewTestWorkflowEnvironment()
			env.SetHeader(&commonpb.Header{Fields: tc.header})
			env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{Interceptor()}})
			env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: tc.id})
			env.RegisterActivity(probeActivity)
			env.RegisterWorkflow(probeChildWorkflow)

			env.ExecuteWorkflow(probeWorkflow)
			requireRefused(t, env.GetWorkflowError(), tc.wantType)
			if tc.wantType == "" {
				var got probe
				require.NoError(t, env.GetWorkflowResult(&got))
				require.Equal(t, probe{Workflow: uid.String(), Activity: uid.String(), Child: uid.String()}, got)
			}
		})
	}
}

// TestInterceptorActivity checks that an activity runs only as a user.
func TestInterceptorActivity(t *testing.T) {
	uid := uuid.Must(uuid.NewV7())

	tests := map[string]struct {
		header   header
		wantType string
	}{
		"no user": {header: header{}, wantType: "ErrMissingRequiredParameter"},
		"a user":  {header: header{HeaderKey: payload(t, uid.String())}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var s testsuite.WorkflowTestSuite
			s.SetContextPropagators([]workflow.ContextPropagator{Propagator()})
			env := s.NewTestActivityEnvironment()
			env.SetHeader(&commonpb.Header{Fields: tc.header})
			env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{Interceptor()}})
			env.RegisterActivity(probeActivity)

			got, err := env.ExecuteActivity(probeActivity)
			requireRefused(t, err, tc.wantType)
			if tc.wantType == "" {
				var seen string
				require.NoError(t, got.Get(&seen))
				require.Equal(t, uid.String(), seen)
			}
		})
	}
}
