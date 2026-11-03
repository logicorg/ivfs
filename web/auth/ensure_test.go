package auth_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/provisioning/common/types/usertypes"
	"github.com/logicorg/ivfs/provisioning/workflows/users"
	"github.com/logicorg/ivfs/web/auth"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

func TestEnsureUserWith(t *testing.T) {
	uid := uuid.Must(uuid.NewV7())

	tests := map[string]struct {
		startErr, runErr error
		wantErr          string
	}{
		"a user":             {},
		"temporal is down":   {startErr: errors.New("connection refused"), wantErr: "connection refused"},
		"the workflow fails": {runErr: errors.New("create database: disk full"), wantErr: "disk full"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &mocks.Client{}
			run := &mocks.WorkflowRun{}
			run.On("Get", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				args.Get(1).(*usertypes.EnsureUserResult).ID = uid
			}).Return(tc.runErr).Maybe()
			var options client.StartWorkflowOptions
			c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.MatchedBy(func(wf any) bool {
				return reflect.ValueOf(wf).Pointer() == reflect.ValueOf(users.EnsureUser).Pointer()
			}), usertypes.EnsureUserParams{Issuer: "https://idp.example", Subject: "alice"}).
				Run(func(args mock.Arguments) { options = args.Get(1).(client.StartWorkflowOptions) }).
				Return(run, tc.startErr).Once()

			got, err := auth.EnsureUserWith(c)(t.Context(), "https://idp.example", "alice")

			c.AssertExpectations(t)
			require.Equal(t, "provisioning", options.TaskQueue)
			require.Equal(t, enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING, options.WorkflowIDConflictPolicy)
			require.Positive(t, options.WorkflowExecutionTimeout)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Equal(t, uuid.Nil, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, uid, got)
		})
	}
}

// TestEnsureUserWith_WorkflowID checks that every sign-in of an identity
// joins one run, that no two identities share one, and that the ID does not
// carry the subject.
func TestEnsureUserWith_WorkflowID(t *testing.T) {
	identities := [][2]string{
		{"https://idp.example", "alice"},
		{"https://idp.example", "alice"},
		{"https://idp.example", "bob"},
		{"https://other.example", "alice"},
		// Without a separator these two would hash the same bytes.
		{"https://idp.example/a", "lice"},
	}
	ids := make([]string, len(identities))
	for i, identity := range identities {
		c := &mocks.Client{}
		run := &mocks.WorkflowRun{}
		run.On("Get", mock.Anything, mock.Anything).Return(nil)
		c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { ids[i] = args.Get(1).(client.StartWorkflowOptions).ID }).
			Return(run, nil)
		_, err := auth.EnsureUserWith(c)(context.Background(), identity[0], identity[1])
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(ids[i], "ensure-user/"), ids[i])
		require.NotContains(t, ids[i], identity[1])
	}

	require.Equal(t, ids[0], ids[1])
	require.Len(t, map[string]bool{ids[0]: true, ids[2]: true, ids[3]: true, ids[4]: true}, 4)
}
