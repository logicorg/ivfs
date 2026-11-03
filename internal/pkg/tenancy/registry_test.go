package tenancy_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uptrace/bun"
)

type RegistryPoolsTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite

	server registry.Server
}

func (s *RegistryPoolsTestSuite) SetupTest() {
	dsn, err := s.Postgres().ConnectionString(s.T().Context(), "sslmode=disable")
	require.NoError(s.T(), err)
	s.server = registry.Server{DSN: dsn}
}

func (s *RegistryPoolsTestSuite) BeforeTest(_, _ string) {
	require.NoError(s.T(), registry.Init(s.T().Context(), s.DB))
}

func (s *RegistryPoolsTestSuite) AfterTest(_, _ string) {
	_, err := s.DB.ExecContext(s.T().Context(), "DROP SCHEMA IF EXISTS iam CASCADE")
	require.NoError(s.T(), err)
}

func Test_RegistryPoolsTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryPoolsTestSuite))
}

func (s *RegistryPoolsTestSuite) Test_RegistryPools_For() {
	tcs := map[string]struct {
		ctx         func(uid uuid.UUID) context.Context
		registered  bool
		provisioned bool
		wantErr     string
	}{
		"a provisioned user gets their initialised database": {
			ctx:         s.as,
			registered:  true,
			provisioned: true,
		},
		"no user": {
			ctx:         func(uuid.UUID) context.Context { return s.T().Context() },
			registered:  true,
			provisioned: true,
			wantErr:     "missing required parameters: user",
		},
		"the nil user": {
			ctx:         func(uuid.UUID) context.Context { return tenancy.WithUser(s.T().Context(), uuid.Nil) },
			registered:  true,
			provisioned: true,
			wantErr:     "missing required parameters: user",
		},
		"an unregistered user": {
			ctx:     s.as,
			wantErr: "resource User not found",
		},
		"an unprovisioned user": {
			ctx:        s.as,
			registered: true,
			wantErr:    "is not provisioned yet",
		},
	}

	for name, tc := range tcs {
		s.Run(name, func() {
			uid := s.newUser(tc.registered, tc.provisioned)
			pools := tenancy.NewRegistryPools(s.DB, s.server)
			defer func() { require.NoError(s.T(), pools.Close()) }()

			db, err := pools.For(tc.ctx(uid))
			if tc.wantErr != "" {
				require.ErrorContains(s.T(), err, tc.wantErr)
				require.Nil(s.T(), db)
				return
			}
			require.NoError(s.T(), err)

			var database string
			require.NoError(s.T(), db.QueryRowContext(s.T().Context(), "SELECT current_database()").Scan(&database))
			require.Equal(s.T(), registry.DatabaseName(uid), database)
			count, err := db.NewSelect().TableExpr("information_schema.schemata").
				Where("schema_name IN (?)", bun.List(model.Schemas)).Count(s.T().Context())
			require.NoError(s.T(), err)
			require.Equal(s.T(), len(model.Schemas), count)
		})
	}
}

// Test_RegistryPools_For_SeparatesUsers stores the same tag for two users:
// in a shared database the second would break its unique name.
func (s *RegistryPoolsTestSuite) Test_RegistryPools_For_SeparatesUsers() {
	pools := tenancy.NewRegistryPools(s.DB, s.server)
	defer func() { require.NoError(s.T(), pools.Close()) }()

	for _, uid := range []uuid.UUID{s.newUser(true, true), s.newUser(true, true)} {
		db, err := pools.For(s.as(uid))
		require.NoError(s.T(), err)
		temporaltesting.Insert(s.T(), db, &model.Tag{Name: "groceries"})

		var tags []model.Tag
		require.NoError(s.T(), db.NewSelect().Model(&tags).Scan(s.T().Context()))
		require.Len(s.T(), tags, 1)
		require.Equal(s.T(), int64(1), tags[0].ID)
	}
}

// Test_RegistryPools_For_RetriesAfterFailure checks that a failed open is not
// cached: a user who signed in halfway through provisioning gets their
// database once it is done.
func (s *RegistryPoolsTestSuite) Test_RegistryPools_For_RetriesAfterFailure() {
	uid := s.newUser(true, false)
	pools := tenancy.NewRegistryPools(s.DB, s.server)
	defer func() { require.NoError(s.T(), pools.Close()) }()

	_, err := pools.For(s.as(uid))
	require.ErrorContains(s.T(), err, "is not provisioned yet")

	_, err = s.DB.NewUpdate().Model((*registry.User)(nil)).
		Set("provisioned_at = current_timestamp").Where("id = ?", uid).Exec(s.T().Context())
	require.NoError(s.T(), err)

	_, err = pools.For(s.as(uid))
	require.NoError(s.T(), err)
}

// Test_RegistryPools_For_Concurrent opens one user's database from many
// activities at once: they share one pool, and their first opens do not race
// each other's InitDatabase.
func (s *RegistryPoolsTestSuite) Test_RegistryPools_For_Concurrent() {
	uid := s.newUser(true, true)
	pools := tenancy.NewRegistryPools(s.DB, s.server)
	defer func() { require.NoError(s.T(), pools.Close()) }()

	const n = 8
	dbs := make([]*bun.DB, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() { dbs[i], errs[i] = pools.For(s.as(uid)) })
	}
	wg.Wait()

	for i := range n {
		require.NoError(s.T(), errs[i])
		require.Same(s.T(), dbs[0], dbs[i])
	}
}

func (s *RegistryPoolsTestSuite) as(uid uuid.UUID) context.Context {
	return tenancy.WithUser(s.T().Context(), uid)
}

// newUser creates a database for a new uid, bare as CreateUserDatabase leaves
// it, and registers the uid when asked to. Both go when the test ends.
func (s *RegistryPoolsTestSuite) newUser(registered, provisioned bool) uuid.UUID {
	uid := uuid.Must(uuid.NewV7())
	name := registry.DatabaseName(uid)
	_, err := s.DB.ExecContext(s.T().Context(), "CREATE DATABASE ?", bun.Ident(name))
	require.NoError(s.T(), err)
	s.T().Cleanup(func() {
		_, err := s.DB.ExecContext(context.Background(), "DROP DATABASE IF EXISTS ? WITH (FORCE)", bun.Ident(name))
		require.NoError(s.T(), err)
	})

	if registered {
		user := &registry.User{ID: uid, Issuer: "https://idp.example", Subject: uid.String()}
		if provisioned {
			now := time.Now()
			user.ProvisionedAt = &now
		}
		_, err := s.DB.NewInsert().Model(user).Exec(s.T().Context())
		require.NoError(s.T(), err)
	}
	return uid
}
