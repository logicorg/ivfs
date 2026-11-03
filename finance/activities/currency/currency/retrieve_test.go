package currency

import (
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_FindCurrencyBy() {
	testcases := []struct {
		name      string
		in        ct.FindCurrencyByParams
		out       *model.Currency
		shouldErr bool
		errStr    string
	}{
		{
			name: "validation error",
			// at least one input field must be non-zero
			in:        ct.FindCurrencyByParams{},
			shouldErr: true,
			errStr:    "validation error",
		},
		{
			name: "Retrieve by a code in another casing",
			in: ct.FindCurrencyByParams{
				Code: " eur ",
			},
			out: testhelpers.Euro(),
		},
		{
			name: "Retrieve by code",
			in: ct.FindCurrencyByParams{
				Code: "EUR",
			},
			out: testhelpers.Euro(),
		},
		{
			name: "Retrieve by name",
			in: ct.FindCurrencyByParams{
				Name: "US Dollar",
			},
			out: testhelpers.USDollar(),
		},
		{
			name: "Retrieve by symbol",
			in: ct.FindCurrencyByParams{
				Symbol: "€",
			},
			out: testhelpers.Euro(),
		},
		{
			name: "Not found",
			in: ct.FindCurrencyByParams{
				Code: "GBP",
			},
			shouldErr: true,
			errStr:    "resource Currency not found",
		},
		{
			name: "Code and name of the same currency",
			in: ct.FindCurrencyByParams{
				Code: "EUR",
				Name: "Euro",
			},
			out: testhelpers.Euro(),
		},
		{
			// The filters are ANDed, so they have to describe one currency.
			name: "Code and name of different currencies",
			in: ct.FindCurrencyByParams{
				Code: "EUR",
				Name: "US Dollar",
			},
			shouldErr: true,
			errStr:    "resource Currency not found",
		},
	}

	s.InsertCurrency(s.T(), testhelpers.Euro())
	s.InsertCurrency(s.T(), testhelpers.USDollar())

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {

			actRes, err := s.Env.ExecuteActivity(s.ca.FindCurrencyBy, tc.in)

			if tc.shouldErr {
				require.Error(t, err)

				require.ErrorContains(t, err, tc.errStr)
			} else {
				require.NoError(t, err)
				var res *model.Currency
				require.NoError(t, actRes.Get(&res))
				require.NotNil(t, res)
				require.Equal(t, tc.out.Code, res.Code)
				require.Equal(t, tc.out.Name, res.Name)
				require.Equal(t, tc.out.Symbol, res.Symbol)
			}
		})
	}
}

// Test_RetrieveCurrencyByID checks that a currency is found by its ID, and
// that an ID with no currency is reported as not found.
func (s *UnitTestSuite) Test_RetrieveCurrencyByID() {
	tests := map[string]struct {
		// id picks the ID to retrieve, given the ID of the Euro fixture.
		id     func(euroID int64) int64
		errStr string
	}{
		"an existing id": {id: func(euroID int64) int64 { return euroID }},
		"a missing id":   {id: func(int64) int64 { return 999 }, errStr: "resource Currency not found"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			euroID := s.InsertCurrency(s.T(), testhelpers.Euro())
			s.T().Cleanup(func() { s.DeleteCurrency(s.T(), euroID) })

			actRes, err := s.Env.ExecuteActivity(s.ca.RetrieveCurrencyByID, tc.id(euroID))

			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
				return
			}
			require.NoError(s.T(), err)
			var res *model.Currency
			require.NoError(s.T(), actRes.Get(&res))
			require.Equal(s.T(), euroID, res.ID)
			require.Equal(s.T(), "EUR", res.Code)
		})
	}
}

// Test_RetrieveCurrencyByCode checks that a currency is found by its code,
// and that a code with no currency is reported as not found.
func (s *UnitTestSuite) Test_RetrieveCurrencyByCode() {
	tests := map[string]struct {
		code   string
		errStr string
	}{
		"an existing code":                   {code: "EUR"},
		"an existing code in another casing": {code: " eur "},
		"a missing code":                     {code: "GBP", errStr: "resource Currency not found"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			euroID := s.InsertCurrency(s.T(), testhelpers.Euro())
			s.T().Cleanup(func() { s.DeleteCurrency(s.T(), euroID) })

			actRes, err := s.Env.ExecuteActivity(s.ca.RetrieveCurrencyByCode, tc.code)

			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
				return
			}
			require.NoError(s.T(), err)
			var res *model.Currency
			require.NoError(s.T(), actRes.Get(&res))
			require.Equal(s.T(), euroID, res.ID)
			require.Equal(s.T(), "EUR", res.Code)
		})
	}
}

// Test_ListCurrencies checks that every currency is listed, and that listing
// an empty table is not an error. The list has no defined order, so only its
// contents are compared.
func (s *UnitTestSuite) Test_ListCurrencies() {
	tests := map[string]struct {
		currencies []*model.Currency
		codes      []string
	}{
		"every currency": {
			currencies: []*model.Currency{testhelpers.Euro(), testhelpers.USDollar(), testhelpers.JapaneseYen()},
			codes:      []string{"EUR", "USD", "JPY"},
		},
		"an empty table": {codes: []string{}},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			for _, c := range tc.currencies {
				id := s.InsertCurrency(s.T(), c)
				s.T().Cleanup(func() { s.DeleteCurrency(s.T(), id) })
			}

			actRes, err := s.Env.ExecuteActivity(s.ca.ListCurrencies)

			require.NoError(s.T(), err)
			var res []*model.Currency
			require.NoError(s.T(), actRes.Get(&res))
			codes := make([]string, len(res))
			for i, c := range res {
				codes[i] = c.Code
			}
			require.ElementsMatch(s.T(), tc.codes, codes)
		})
	}
}

// Test_CurrencyExists checks that the existence of a currency is reported by
// its code.
func (s *UnitTestSuite) Test_CurrencyExists() {
	tests := map[string]struct {
		code   string
		exists bool
	}{
		"an existing code":                   {code: "EUR", exists: true},
		"an existing code in another casing": {code: " eur ", exists: true},
		"a missing code":                     {code: "GBP", exists: false},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			euroID := s.InsertCurrency(s.T(), testhelpers.Euro())
			s.T().Cleanup(func() { s.DeleteCurrency(s.T(), euroID) })

			actRes, err := s.Env.ExecuteActivity(s.ca.CurrencyExists, tc.code)

			require.NoError(s.T(), err)
			var exists bool
			require.NoError(s.T(), actRes.Get(&exists))
			require.Equal(s.T(), tc.exists, exists)
		})
	}
}
