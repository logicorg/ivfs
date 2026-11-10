package temporaltesting

import (
	"database/sql"
	"log"
	"testing"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.temporal.io/sdk/testsuite"
)

// TemporalDBUnitTestSuite base suite to handle testing temporal activities interacting with a DB
type TemporalDBUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	postgresContainer *postgres.PostgresContainer
	DB                *bun.DB

	Env *testsuite.TestActivityEnvironment
}

type Suite interface {
	T() *testing.T
	GetDB() *bun.DB
}

func (s *TemporalDBUnitTestSuite) GetDB() *bun.DB {
	return s.DB
}

// SetupSuite setup the test
func (s *TemporalDBUnitTestSuite) SetupSuite() {
	ctx := s.T().Context()
	dbName := "database"
	dbUser := "user"
	dbPassword := "password"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Printf("failed to start container: %s", err)
		return
	}
	s.postgresContainer = postgresContainer

	connStr, err := s.postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(s.T(), err, "failed to get connection string")

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(connStr)))
	s.DB = bun.NewDB(sqldb, pgdialect.New())

	sqllib.InitDatabaseT(s.T(), s.DB)
}

// TearDownSuite runs the test teardown operations
func (s *TemporalDBUnitTestSuite) TearDownSuite() {
	if err := testcontainers.TerminateContainer(s.postgresContainer); err != nil {
		s.T().Logf("failed to terminate container: %s", err)
	}
	if err := s.DB.Close(); err != nil {
		s.T().Log("failed to close database connection", err)
	}
}

// SetupTestEnvironment setup the test environment by creating a test activity environment
func (s *TemporalDBUnitTestSuite) SetupTestEnvironment() {
	s.Env = s.NewTestActivityEnvironment()
}

// Postgres returns the postgres test container
func (s *TemporalDBUnitTestSuite) Postgres() *postgres.PostgresContainer {
	return s.postgresContainer
}

// DropAllTables drops all tables from the DB
func (s *TemporalDBUnitTestSuite) DropAllTables() {
	for _, schema := range model.Schemas {
		query := "DROP SCHEMA IF EXISTS " + schema + " CASCADE"
		_, err := s.DB.Exec(query)
		if err != nil {
			s.T().Logf("failed to drop schema %s: %s", schema, err)
		}
	}
}

func (s *TemporalDBUnitTestSuite) Count[O any]() int {
	s.T().Helper()

	count, err := s.DB.NewSelect().Model((*O)(nil)).Count(s.T().Context())
	require.NoError(s.T(), err)
	return int(count)
}

func (s *TemporalDBUnitTestSuite) CountWhere[O any](query string, args ...any) int {
	s.T().Helper()

	count, err := s.DB.NewSelect().Model((*O)(nil)).Where(query, args...).Count(s.T().Context())
	require.NoError(s.T(), err)
	return int(count)
}

func CountWhere[O any](t *testing.T, db bun.IDB, query string, args ...any) int {
	t.Helper()

	count, err := db.NewSelect().Model((*O)(nil)).Where(query, args...).Count(t.Context())
	require.NoError(t, err)
	return int(count)
}
