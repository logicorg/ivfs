package users

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/provisioning/activities/user"
	"github.com/logicorg/ivfs/provisioning/common/types/usertypes"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type EnsureUserUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestEnsureUserUnitTestSuite(t *testing.T) {
	suite.Run(t, new(EnsureUserUnitTestSuite))
}

func (s *EnsureUserUnitTestSuite) Test_EnsureUser() {
	uid := uuid.MustParse("0199c4a2-7b1e-7c3a-9f00-000000000001")
	params := usertypes.EnsureUserParams{Issuer: "https://idp.example", Subject: "alice"}

	tests := []struct {
		name         string
		inserted     *usertypes.InsertUserResult
		insertErr    error
		createErr    error
		provisionErr error
		// wantCreates and wantProvisions count attempts, retries included.
		wantCreates    int
		wantProvisions int
		want           *usertypes.EnsureUserResult
		wantErr        string
	}{
		{
			name:           "a new user is provisioned",
			inserted:       &usertypes.InsertUserResult{ID: uid},
			wantCreates:    1,
			wantProvisions: 1,
			want:           &usertypes.EnsureUserResult{ID: uid},
		},
		{
			name:     "a provisioned user skips provisioning",
			inserted: &usertypes.InsertUserResult{ID: uid, Provisioned: true},
			want:     &usertypes.EnsureUserResult{ID: uid},
		},
		{
			name:      "an invalid identity is not retried",
			insertErr: internalerrors.ValidationError{Err: errors.New("subject is blank")},
			wantErr:   "subject is blank",
		},
		{
			name:        "an unknown uid stops before provisioning",
			inserted:    &usertypes.InsertUserResult{ID: uid},
			createErr:   internalerrors.ErrNotFound{ResourceType: "User", ID: uid},
			wantCreates: 1,
			wantErr:     "resource User not found",
		},
		{
			name:           "a database failure is retried",
			inserted:       &usertypes.InsertUserResult{ID: uid},
			provisionErr:   errors.New("connection refused"),
			wantCreates:    1,
			wantProvisions: 5,
			wantErr:        "connection refused",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := s.NewTestWorkflowEnvironment()
			var a *user.Activities
			env.OnActivity(a.InsertUser, mock.Anything, usertypes.InsertUserParams(params)).Return(tt.inserted, tt.insertErr)
			env.OnActivity(a.CreateUserDatabase, mock.Anything, uid).Return(tt.createErr)
			env.OnActivity(a.ProvisionUser, mock.Anything, uid).Return(tt.provisionErr)

			env.ExecuteWorkflow(EnsureUser, params)

			s.True(env.IsWorkflowCompleted())
			env.AssertActivityNumberOfCalls(s.T(), "InsertUser", 1)
			env.AssertActivityNumberOfCalls(s.T(), "CreateUserDatabase", tt.wantCreates)
			env.AssertActivityNumberOfCalls(s.T(), "ProvisionUser", tt.wantProvisions)
			if tt.wantErr != "" {
				s.ErrorContains(env.GetWorkflowError(), tt.wantErr)
				return
			}
			s.Require().NoError(env.GetWorkflowError())
			var got usertypes.EnsureUserResult
			s.Require().NoError(env.GetWorkflowResult(&got))
			s.Equal(tt.want, &got)
		})
	}
}
