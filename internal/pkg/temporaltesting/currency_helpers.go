package temporaltesting

import (
	"testing"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

// InsertCurrency is a helper function to insert a currency and return its ID
func (s *TemporalDBUnitTestSuite) InsertCurrency(t *testing.T, currency *model.Currency) int64 {
	t.Helper()

	_, err := s.DB.NewInsert().Model(currency).Returning("id").Exec(t.Context())
	require.NoError(s.T(), err)
	require.NotZero(s.T(), currency.ID)
	return currency.ID
}

// DeleteCurrency is a helper function to delete a currency with a given ID from the DB
func (s *TemporalDBUnitTestSuite) DeleteCurrency(t *testing.T, id int64) {
	t.Helper()

	_, err := s.DB.NewDelete().Model((*model.Currency)(nil)).Where("id = ?", id).Exec(t.Context())
	require.NoError(s.T(), err)
}
