package user

import (
	"testing"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uptrace/bun"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite

	activities *Activities
}

func (s *UnitTestSuite) SetupTest() {
	s.SetupTestEnvironment()
	dsn, err := s.Postgres().ConnectionString(s.T().Context(), "sslmode=disable")
	require.NoError(s.T(), err)
	s.activities = &Activities{Registry: s.DB, Server: registry.Server{DSN: dsn}}
	s.Env.RegisterActivity(s.activities)
}

func (s *UnitTestSuite) BeforeTest(_, _ string) {
	require.NoError(s.T(), registry.Init(s.T().Context(), s.DB))
}

func (s *UnitTestSuite) AfterTest(_, _ string) {
	_, err := s.DB.ExecContext(s.T().Context(), "DROP SCHEMA IF EXISTS iam CASCADE")
	require.NoError(s.T(), err)
}

func Test_UnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

// createDatabase is a fixture: the activities' tests set up databases with
// SQL rather than with each other.
func (s *UnitTestSuite) createDatabase(name string) {
	_, err := s.DB.ExecContext(s.T().Context(), "CREATE DATABASE ?", bun.Ident(name))
	require.NoError(s.T(), err)
}

// dropDatabase removes name, and the connections a failed test left on it.
func (s *UnitTestSuite) dropDatabase(name string) {
	_, err := s.DB.ExecContext(s.T().Context(), "DROP DATABASE IF EXISTS ? WITH (FORCE)", bun.Ident(name))
	require.NoError(s.T(), err)
}

func (s *UnitTestSuite) databaseExists(name string) bool {
	exists, err := s.DB.NewSelect().TableExpr("pg_database").Where("datname = ?", name).Exists(s.T().Context())
	require.NoError(s.T(), err)
	return exists
}

func (s *UnitTestSuite) retrieveUser(user *registry.User) {
	require.NoError(s.T(), s.DB.NewSelect().Model(user).WherePK().Scan(s.T().Context()))
}
