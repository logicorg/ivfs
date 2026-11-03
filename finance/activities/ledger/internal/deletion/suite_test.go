package deletion

import (
	"testing"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uptrace/bun"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite
}

func (s *UnitTestSuite) BeforeTest(_, _ string) { sqllib.InitDatabaseT(s.T(), s.DB) }
func (s *UnitTestSuite) AfterTest(_, _ string)  { s.DropAllTables() }
func Test_UnitTestSuite(t *testing.T)           { suite.Run(t, new(UnitTestSuite)) }

// inRolledBackTx runs a case in a transaction that is rolled back afterwards,
// which removes whatever the case wrote.
func (s *UnitTestSuite) inRolledBackTx(run func(tx bun.Tx)) {
	s.T().Helper()

	tx, err := s.DB.BeginTx(s.T().Context(), nil)
	require.NoError(s.T(), err)
	defer func() { _ = tx.Rollback() }()
	run(tx)
}

// insertTx inserts a fixture inside the case's transaction.
func insertTx[M any](t *testing.T, tx bun.Tx, m *M) {
	t.Helper()

	_, err := tx.NewInsert().Model(m).Returning("id").Exec(t.Context())
	require.NoError(t, err)
}
