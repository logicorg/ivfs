package currency

import (
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_Delete() {
	currencyID := s.InsertCurrency(s.T(), testhelpers.Euro())

	_, err := s.Env.ExecuteActivity(s.ca.DeleteCurrency, ct.DeleteCurrencyParams{
		ID: currencyID,
	})
	require.NoError(s.T(), err)

	found, err := model.FindCurrencyByIDQuery(s.DB, currencyID).Exists(s.T().Context())
	require.NoError(s.T(), err, "failed to check if currency exists")
	require.False(s.T(), found, "expected currency to be deleted")
}

func (s *UnitTestSuite) currencyExists(id int64) bool {
	s.T().Helper()

	found, err := model.FindCurrencyByIDQuery(s.DB, id).Exists(s.T().Context())
	require.NoError(s.T(), err)
	return found
}

// Test_Delete_Selectors checks how the currency to delete is selected: by
// code, and by ID winning over a code given alongside it. Only the Euro is
// ever selected, so the US dollar has to survive every case.
func (s *UnitTestSuite) Test_Delete_Selectors() {
	tests := map[string]struct {
		// params builds the parameters, given the ID of the Euro fixture.
		params func(euroID int64) ct.DeleteCurrencyParams
	}{
		"by code": {
			params: func(int64) ct.DeleteCurrencyParams { return ct.DeleteCurrencyParams{Code: "EUR"} },
		},
		"by code in another casing": {
			params: func(int64) ct.DeleteCurrencyParams { return ct.DeleteCurrencyParams{Code: " eur "} },
		},
		"the id wins over the code": {
			params: func(euroID int64) ct.DeleteCurrencyParams { return ct.DeleteCurrencyParams{ID: euroID, Code: "USD"} },
		},
	}

	for name, tc := range tests {
		euroID := s.InsertCurrency(s.T(), testhelpers.Euro())
		dollarID := s.InsertCurrency(s.T(), testhelpers.USDollar())

		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.ca.DeleteCurrency, tc.params(euroID))

			require.NoError(s.T(), err)
			require.False(s.T(), s.currencyExists(euroID))
			require.True(s.T(), s.currencyExists(dollarID))
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Currency)(nil), euroID)
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Currency)(nil), dollarID)
	}
}

// Test_Delete_MissingParameters checks that a delete without an ID or a code
// is rejected rather than deleting anything.
func (s *UnitTestSuite) Test_Delete_MissingParameters() {
	currencyID := s.InsertCurrency(s.T(), testhelpers.Euro())

	_, err := s.Env.ExecuteActivity(s.ca.DeleteCurrency, ct.DeleteCurrencyParams{})

	require.ErrorContains(s.T(), err, "missing required parameters")
	require.True(s.T(), s.currencyExists(currencyID))
}

// Test_Delete_Idempotent checks that deleting a currency that is already gone
// succeeds, so a retried delete does not fail.
func (s *UnitTestSuite) Test_Delete_Idempotent() {
	currencyID := s.InsertCurrency(s.T(), testhelpers.Euro())

	_, err := s.Env.ExecuteActivity(s.ca.DeleteCurrency, ct.DeleteCurrencyParams{ID: currencyID})
	require.NoError(s.T(), err)
	_, err = s.Env.ExecuteActivity(s.ca.DeleteCurrency, ct.DeleteCurrencyParams{ID: currencyID})
	require.NoError(s.T(), err)

	require.False(s.T(), s.currencyExists(currencyID))
}

// A currency that ledger accounts are denominated in cannot be deleted.
func (s *UnitTestSuite) Test_Delete_InUse() {
	currencyID := s.InsertCurrency(s.T(), testhelpers.Euro())
	temporaltesting.Insert(s.T(), s.DB, &model.LedgerAccount{CurrencyID: currencyID, Name: "cash", Kind: model.LedgerAccountAsset})

	_, err := s.Env.ExecuteActivity(s.ca.DeleteCurrency, ct.DeleteCurrencyParams{ID: currencyID})

	var appErr *temporal.ApplicationError
	require.ErrorAs(s.T(), err, &appErr)
	require.Equal(s.T(), "ErrResourceConflict", appErr.Type())

	found, err := model.FindCurrencyByIDQuery(s.DB, currencyID).Exists(s.T().Context())
	require.NoError(s.T(), err)
	require.True(s.T(), found)
}
