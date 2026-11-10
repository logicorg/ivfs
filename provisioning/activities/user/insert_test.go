package user

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/logicorg/ivfs/provisioning/common/types/usertypes"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_InsertUser() {
	provisionedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := map[string]struct {
		// existing is the user already registered, if any.
		existing        *registry.User
		params          usertypes.InsertUserParams
		wantProvisioned bool
		wantErr         string
	}{
		"a new identity": {
			params: usertypes.InsertUserParams{Issuer: "https://idp.test", Subject: "alice"},
		},
		"an unprovisioned user": {
			existing: &registry.User{ID: uuid.Must(uuid.NewV7()), Issuer: "https://idp.test", Subject: "alice"},
			params:   usertypes.InsertUserParams{Issuer: "https://idp.test", Subject: "alice"},
		},
		"a provisioned user": {
			existing:        &registry.User{ID: uuid.Must(uuid.NewV7()), Issuer: "https://idp.test", Subject: "alice", ProvisionedAt: &provisionedAt},
			params:          usertypes.InsertUserParams{Issuer: "https://idp.test", Subject: "alice"},
			wantProvisioned: true,
		},
		"a blank issuer": {
			params:  usertypes.InsertUserParams{Issuer: " ", Subject: "alice"},
			wantErr: "validation",
		},
		"a missing subject": {
			params:  usertypes.InsertUserParams{Issuer: "https://idp.test"},
			wantErr: "validation",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			if tc.existing != nil {
				temporaltesting.Insert(s.T(), s.DB, tc.existing)
				defer temporaltesting.DeleteObj(s.T(), s.DB, tc.existing)
			}

			activityResult, err := s.Env.ExecuteActivity(s.activities.InsertUser, tc.params)
			if tc.wantErr != "" {
				require.ErrorContains(s.T(), err, tc.wantErr)
				return
			}
			require.NoError(s.T(), err)
			var result usertypes.InsertUserResult
			require.NoError(s.T(), activityResult.Get(&result))

			stored := &registry.User{ID: result.ID}
			s.retrieveUser(stored)
			defer temporaltesting.DeleteObj(s.T(), s.DB, stored)

			if tc.existing != nil {
				require.Equal(s.T(), tc.existing.ID, result.ID)
			} else {
				require.Equal(s.T(), uuid.Version(7), result.ID.Version())
			}
			require.Equal(s.T(), tc.wantProvisioned, result.Provisioned)
			require.Equal(s.T(), tc.params.Issuer, stored.Issuer)
			require.Equal(s.T(), tc.params.Subject, stored.Subject)
		})
	}
}

func (s *UnitTestSuite) Test_InsertUser_Idempotency() {
	params := usertypes.InsertUserParams{Issuer: "https://idp.test", Subject: "alice"}

	var results [2]usertypes.InsertUserResult
	for n := range results {
		activityResult, err := s.Env.ExecuteActivity(s.activities.InsertUser, params)
		require.NoError(s.T(), err)
		require.NoError(s.T(), activityResult.Get(&results[n]))
	}

	require.Equal(s.T(), results[0], results[1])
	require.Equal(s.T(), 1, s.Count[registry.User]())
}

// Test_InsertUser_Concurrent checks that logins of the same identity racing
// each other all get the one user, since each would otherwise provision a
// database of its own.
func (s *UnitTestSuite) Test_InsertUser_Concurrent() {
	params := usertypes.InsertUserParams{Issuer: "https://idp.test", Subject: "alice"}

	var results [8]*usertypes.InsertUserResult
	var errs [8]error
	var wg sync.WaitGroup
	for n := range results {
		wg.Go(func() {
			results[n], errs[n] = s.activities.InsertUser(s.T().Context(), params)
		})
	}
	wg.Wait()

	for n := range results {
		require.NoError(s.T(), errs[n])
		require.Equal(s.T(), results[0].ID, results[n].ID)
	}
	count, err := s.DB.NewSelect().Model((*registry.User)(nil)).Count(s.T().Context())
	require.NoError(s.T(), err)
	require.EqualValues(s.T(), 1, count)
}
