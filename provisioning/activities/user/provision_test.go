package user

import (
	"time"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func (s *UnitTestSuite) Test_ProvisionUser() {
	provisionedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := map[string]struct {
		registered    bool
		provisionedAt *time.Time
		// database is whether the user's database exists.
		database   bool
		wantSchema bool
		wantErr    string
	}{
		"a user with a database": {registered: true, database: true, wantSchema: true},
		// The database is missing, so the activity can only succeed by
		// leaving it alone.
		"a provisioned user":        {registered: true, provisionedAt: &provisionedAt},
		"a user without a database": {registered: true, wantErr: "does not exist"},
		"an unregistered uid":       {database: true, wantErr: "resource User not found"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			user := &registry.User{ID: uuid.Must(uuid.NewV7()), Issuer: "https://idp.test", Subject: "alice", ProvisionedAt: tc.provisionedAt}
			if tc.registered {
				temporaltesting.Insert(s.T(), s.DB, user)
				defer temporaltesting.DeleteObj(s.T(), s.DB, user)
			}
			database := registry.DatabaseName(user.ID)
			if tc.database {
				s.createDatabase(database)
			}
			defer s.dropDatabase(database)

			_, err := s.Env.ExecuteActivity(s.activities.ProvisionUser, user.ID)
			if tc.wantErr != "" {
				require.ErrorContains(s.T(), err, tc.wantErr)
			} else {
				require.NoError(s.T(), err)
			}
			if tc.database {
				require.Equal(s.T(), tc.wantSchema, s.hasSchemas(database))
			}
			if !tc.registered {
				return
			}

			stored := &registry.User{ID: user.ID}
			s.retrieveUser(stored)
			switch {
			case tc.provisionedAt != nil:
				require.True(s.T(), tc.provisionedAt.Equal(*stored.ProvisionedAt))
			case tc.wantErr != "":
				require.Nil(s.T(), stored.ProvisionedAt)
			default:
				require.NotNil(s.T(), stored.ProvisionedAt)
			}
		})
	}
}

func (s *UnitTestSuite) Test_ProvisionUser_Idempotency() {
	user := &registry.User{ID: uuid.Must(uuid.NewV7()), Issuer: "https://idp.test", Subject: "alice"}
	temporaltesting.Insert(s.T(), s.DB, user)
	defer temporaltesting.DeleteObj(s.T(), s.DB, user)
	database := registry.DatabaseName(user.ID)
	s.createDatabase(database)
	defer s.dropDatabase(database)

	var provisioned [2]*time.Time
	for n := range provisioned {
		_, err := s.Env.ExecuteActivity(s.activities.ProvisionUser, user.ID)
		require.NoError(s.T(), err)
		stored := &registry.User{ID: user.ID}
		s.retrieveUser(stored)
		provisioned[n] = stored.ProvisionedAt
	}

	require.NotNil(s.T(), provisioned[0])
	require.True(s.T(), provisioned[0].Equal(*provisioned[1]))
	require.True(s.T(), s.hasSchemas(database))
}

// hasSchemas reports whether every schema of a user's database is in name.
func (s *UnitTestSuite) hasSchemas(name string) bool {
	db := s.activities.Server.Open(name)
	defer db.Close()
	count, err := db.NewSelect().TableExpr("information_schema.schemata").
		Where("schema_name IN (?)", bun.List(model.Schemas)).
		Count(s.T().Context())
	require.NoError(s.T(), err)
	return count == len(model.Schemas)
}
