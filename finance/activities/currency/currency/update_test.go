package currency

import (
	"fmt"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_Update() {
	euro := testhelpers.Euro()

	// testacases are the test case to run for the update activity.
	// Before each individual test case is run, the Euro currency is
	// insterted, and it is deleted at the end of each test.
	// Set hydrateInputWithID to true in the test cases for which you
	// want the ID of such currency to be added to the input parameters.
	// Leave false/unset for those test cases in which you do not want
	// this ID to be added to the input params (e.g. if you want to test
	// a failure if an invalid ID is supplied).
	testcases := []struct {
		name  string
		input ct.UpdateCurrencyParams
		// hydrateInputWithID set to true when you want the ID of the currency
		// inserted before each test case to be set in the input parameters struct.
		hydrateInputWithID bool
		expected           *model.Currency
		shouldError        bool
		errString          string
	}{
		{
			name:               "Update name",
			hydrateInputWithID: true,
			input: ct.UpdateCurrencyParams{
				Name: ptr.NewPtr("European Euro"),
			},
			expected: &model.Currency{
				Code:      euro.Code,
				Name:      "European Euro",
				Symbol:    euro.Symbol,
				Precision: euro.Precision,
			},
		},
		{
			name:               "Update symbol",
			hydrateInputWithID: true,
			input: ct.UpdateCurrencyParams{
				Symbol: ptr.NewPtr("E"),
			},
			expected: &model.Currency{
				Code:      euro.Code,
				Name:      euro.Name,
				Symbol:    "E",
				Precision: euro.Precision,
			},
		},
		{
			name:               "Update code",
			hydrateInputWithID: true,
			input: ct.UpdateCurrencyParams{
				Code: ptr.NewPtr("EUR-1"),
			},
			expected: &model.Currency{
				Code:      "EUR-1",
				Name:      euro.Name,
				Symbol:    euro.Symbol,
				Precision: euro.Precision,
			},
		},
		{
			name:               "Update code in another casing",
			hydrateInputWithID: true,
			input: ct.UpdateCurrencyParams{
				Code: ptr.NewPtr(" gbp "),
			},
			expected: &model.Currency{
				Code:      "GBP",
				Name:      euro.Name,
				Symbol:    euro.Symbol,
				Precision: euro.Precision,
			},
		},
		{
			name:        "Not found",
			input:       ct.UpdateCurrencyParams{ID: 999},
			shouldError: true,
			errString:   fmt.Sprintf("resource Currency not found with ID: %d", 999),
		},
		{
			name:        "Missing ID",
			input:       ct.UpdateCurrencyParams{},
			shouldError: true,
			errString:   "missing required parameters: ID",
		},
	}

	for _, tc := range testcases {
		id := s.InsertCurrency(s.T(), testhelpers.Euro())

		s.T().Run(tc.name, func(t *testing.T) {
			if tc.hydrateInputWithID {
				tc.input.ID = id
			}
			_, err := s.Env.ExecuteActivity(s.ca.UpdateCurrency, tc.input)

			if tc.shouldError {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errString)
			} else {
				require.NoError(t, err)

				// Verify the update in the database
				var updatedCurrency model.Currency
				err := s.DB.NewSelect().Model(&updatedCurrency).Where("id = ?", tc.input.ID).Scan(t.Context(), &updatedCurrency)
				require.NoError(t, err)

				require.Equal(t, tc.expected.Code, updatedCurrency.Code)
				require.Equal(t, tc.expected.Name, updatedCurrency.Name)
				require.Equal(t, tc.expected.Symbol, updatedCurrency.Symbol)
				require.Equal(t, tc.expected.Precision, updatedCurrency.Precision)
			}
		})

		s.DeleteCurrency(s.T(), id)
	}
}

// Test_Update_Idempotent checks that applying the same update twice succeeds
// both times and leaves the currency as the first call did.
func (s *UnitTestSuite) Test_Update_Idempotent() {
	id := s.InsertCurrency(s.T(), testhelpers.Euro())
	params := ct.UpdateCurrencyParams{ID: id, Name: ptr.NewPtr("European Euro")}

	_, err := s.Env.ExecuteActivity(s.ca.UpdateCurrency, params)
	require.NoError(s.T(), err)
	_, err = s.Env.ExecuteActivity(s.ca.UpdateCurrency, params)
	require.NoError(s.T(), err)

	var stored model.Currency
	require.NoError(s.T(), model.FindCurrencyByIDQuery(s.DB, id).Scan(s.T().Context(), &stored))
	require.Equal(s.T(), "European Euro", stored.Name)
	require.Equal(s.T(), "EUR", stored.Code)
	require.Equal(s.T(), "€", stored.Symbol)
}

// Test_Update_Taken checks that giving a currency the code or the name of
// another one is a conflict, and leaves both unchanged.
func (s *UnitTestSuite) Test_Update_Taken() {
	tests := map[string]ct.UpdateCurrencyParams{
		"the code of another currency": {Code: ptr.NewPtr("USD")},
		"the name of another currency": {Name: ptr.NewPtr("US Dollar")},
	}

	for name, params := range tests {
		s.Run(name, func() {
			euroID := s.InsertCurrency(s.T(), testhelpers.Euro())
			dollarID := s.InsertCurrency(s.T(), testhelpers.USDollar())
			s.T().Cleanup(func() {
				s.DeleteCurrency(s.T(), euroID)
				s.DeleteCurrency(s.T(), dollarID)
			})
			params.ID = euroID

			_, err := s.Env.ExecuteActivity(s.ca.UpdateCurrency, params)

			require.ErrorContains(s.T(), err, "resource Currency conflict")
			var stored model.Currency
			require.NoError(s.T(), model.FindCurrencyByIDQuery(s.DB, euroID).Scan(s.T().Context(), &stored))
			require.Equal(s.T(), "EUR", stored.Code)
			require.Equal(s.T(), "Euro", stored.Name)
		})
	}
}
