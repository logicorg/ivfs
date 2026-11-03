package user

import (
	"sync"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_CreateUserDatabase() {
	tests := map[string]struct {
		registered bool
		// existing is whether the database is there before the activity runs.
		existing bool
		nilUID   bool
		wantErr  string
	}{
		"a registered user":                         {registered: true},
		"a database already there":                  {registered: true, existing: true},
		"an unregistered uid":                       {wantErr: "resource User not found"},
		"an unregistered uid whose database exists": {existing: true, wantErr: "resource User not found"},
		"the nil uid":                               {nilUID: true, wantErr: "missing required parameters: uid"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			uid := uuid.Must(uuid.NewV7())
			if tc.nilUID {
				uid = uuid.Nil
			}
			if tc.registered {
				user := &registry.User{ID: uid, Issuer: "https://idp.test", Subject: "alice"}
				temporaltesting.Insert(s.T(), s.DB, user)
				defer temporaltesting.DeleteObj(s.T(), s.DB, user)
			}
			database := registry.DatabaseName(uid)
			if tc.existing {
				s.createDatabase(database)
			}
			defer s.dropDatabase(database)

			_, err := s.Env.ExecuteActivity(s.activities.CreateUserDatabase, uid)
			if tc.wantErr != "" {
				require.ErrorContains(s.T(), err, tc.wantErr)
				require.Equal(s.T(), tc.existing, s.databaseExists(database))
				return
			}
			require.NoError(s.T(), err)
			require.True(s.T(), s.databaseExists(database))
		})
	}
}

func (s *UnitTestSuite) Test_CreateUserDatabase_Idempotency() {
	user := &registry.User{ID: uuid.Must(uuid.NewV7()), Issuer: "https://idp.test", Subject: "alice"}
	temporaltesting.Insert(s.T(), s.DB, user)
	defer temporaltesting.DeleteObj(s.T(), s.DB, user)
	database := registry.DatabaseName(user.ID)
	defer s.dropDatabase(database)

	for range 2 {
		_, err := s.Env.ExecuteActivity(s.activities.CreateUserDatabase, user.ID)
		require.NoError(s.T(), err)
	}
	require.True(s.T(), s.databaseExists(database))
}

// Test_CreateUserDatabase_Concurrent checks that creates racing past the
// check against pg_database all succeed, whichever of them creates it.
func (s *UnitTestSuite) Test_CreateUserDatabase_Concurrent() {
	user := &registry.User{ID: uuid.Must(uuid.NewV7()), Issuer: "https://idp.test", Subject: "alice"}
	temporaltesting.Insert(s.T(), s.DB, user)
	defer temporaltesting.DeleteObj(s.T(), s.DB, user)
	database := registry.DatabaseName(user.ID)
	defer s.dropDatabase(database)

	var errs [4]error
	var wg sync.WaitGroup
	for n := range errs {
		wg.Go(func() {
			errs[n] = s.activities.CreateUserDatabase(s.T().Context(), user.ID)
		})
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(s.T(), err)
	}
	require.True(s.T(), s.databaseExists(database))
}
