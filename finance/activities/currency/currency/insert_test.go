package currency

import (
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_Insert() {
	id2 := s.InsertCurrency(s.T(), testhelpers.USDollar())
	id1 := s.InsertCurrency(s.T(), testhelpers.Euro())
	s.InsertCurrency(s.T(), &model.Currency{
		Code:   "TST",
		Name:   "Test",
		Symbol: "T",
	})
	s.DeleteCurrency(s.T(), id1)
	s.DeleteCurrency(s.T(), id2)

	actRes, err := s.Env.ExecuteActivity(s.ca.InsertCurrency, ct.InsertCurrencyParams{
		Code:   "EUR",
		Name:   "Euro",
		Symbol: "€",
	})
	require.NoError(s.T(), err)

	var res *ct.InsertCurrencyResult
	err = actRes.Get(&res)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), res)
	require.NotEqual(s.T(), 0, res.CurrencyID, "expected currency ID to be non-zero")
}

func (s *UnitTestSuite) insertCurrency(params ct.InsertCurrencyParams) ct.InsertCurrencyResult {
	s.T().Helper()

	actRes, err := s.Env.ExecuteActivity(s.ca.InsertCurrency, params)
	require.NoError(s.T(), err)
	var res ct.InsertCurrencyResult
	require.NoError(s.T(), actRes.Get(&res))
	return res
}

func (s *UnitTestSuite) countCurrencies() int {
	s.T().Helper()

	count, err := s.DB.NewSelect().Model((*model.Currency)(nil)).Count(s.T().Context())
	require.NoError(s.T(), err)
	return count
}

// Test_Insert_Idempotent checks that a retry returns the currency of the
// first call and changes nothing, even when the retried request differs: a
// currency is identified by its code, and a retry does not reconcile the
// other fields against the stored row.
func (s *UnitTestSuite) Test_Insert_Idempotent() {
	params := ct.InsertCurrencyParams{Code: "EUR", Name: "Euro", Symbol: "€"}

	created := s.insertCurrency(params)
	retried := s.insertCurrency(params)

	require.Equal(s.T(), created, retried)
	require.Equal(s.T(), 1, s.countCurrencies())

	different := s.insertCurrency(ct.InsertCurrencyParams{Code: "EUR", Name: "European Euro", Symbol: "E", Precision: ptr.NewPtr(3)})

	require.Equal(s.T(), created, different)
	require.Equal(s.T(), 1, s.countCurrencies())
	var stored model.Currency
	require.NoError(s.T(), model.FindCurrencyByIDQuery(s.DB, created.CurrencyID).Scan(s.T().Context(), &stored))
	require.Equal(s.T(), "Euro", stored.Name)
	require.Equal(s.T(), "€", stored.Symbol)
	require.Equal(s.T(), 2, stored.Precision)
}

// Test_Insert_Rejected checks that a currency that cannot be inserted is
// rejected and nothing is written. A new code with the name of an existing
// currency is a conflict: the name is unique too, but the activity only looks
// the currency up by code.
func (s *UnitTestSuite) Test_Insert_Rejected() {
	tests := map[string]struct {
		params ct.InsertCurrencyParams
		errStr string
	}{
		"no code":                      {params: ct.InsertCurrencyParams{Name: "Pound", Symbol: "£"}, errStr: "missing required parameters: Code"},
		"no name":                      {params: ct.InsertCurrencyParams{Code: "GBP", Symbol: "£"}, errStr: "missing required parameters: Name"},
		"the name of another currency": {params: ct.InsertCurrencyParams{Code: "EUX", Name: "Euro", Symbol: "€"}, errStr: "resource Currency conflict"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			euroID := s.InsertCurrency(s.T(), testhelpers.Euro())
			s.T().Cleanup(func() { s.DeleteCurrency(s.T(), euroID) })

			_, err := s.Env.ExecuteActivity(s.ca.InsertCurrency, tc.params)

			require.ErrorContains(s.T(), err, tc.errStr)
			require.Equal(s.T(), 1, s.countCurrencies(), "a rejected currency must not be written")
		})
	}
}

func (s *UnitTestSuite) Test_Insert_Precision() {
	tests := map[string]struct {
		code      string
		precision *int
		expected  int
	}{
		"defaults to two decimals":  {code: "AAA", precision: nil, expected: 2},
		"explicit precision":        {code: "BBB", precision: ptr.NewPtr(3), expected: 3},
		"zero decimals is not lost": {code: "CCC", precision: ptr.NewPtr(0), expected: 0},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			actRes, err := s.Env.ExecuteActivity(s.ca.InsertCurrency, ct.InsertCurrencyParams{
				Code: tc.code, Name: name, Symbol: "T", Precision: tc.precision,
			})
			require.NoError(s.T(), err)
			var res ct.InsertCurrencyResult
			require.NoError(s.T(), actRes.Get(&res))

			var stored model.Currency
			require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", res.CurrencyID).Scan(s.T().Context()))
			require.Equal(s.T(), tc.expected, stored.Precision)
		})
	}
}

// Test_Insert_NormalisesCode checks that a code is stored in its one canonical
// form, trimmed and upper-cased (ADR 0005).
func (s *UnitTestSuite) Test_Insert_NormalisesCode() {
	tests := map[string]string{
		"lower case":        "eur",
		"mixed case":        "Eur",
		"surrounding space": " EUR ",
	}

	for name, code := range tests {
		var created int64

		s.Run(name, func() {
			result := s.insertCurrency(ct.InsertCurrencyParams{Code: code, Name: "Euro", Symbol: "€"})
			created = result.CurrencyID

			var stored model.Currency
			require.NoError(s.T(), model.FindCurrencyByIDQuery(s.DB, created).Scan(s.T().Context(), &stored))
			require.Equal(s.T(), "EUR", stored.Code)
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Currency)(nil), created)
	}
}

// Test_Insert_RetrievesExistingByCode checks that a currency is retrieved by
// its code in any casing or spacing, rather than inserted a second time.
func (s *UnitTestSuite) Test_Insert_RetrievesExistingByCode() {
	tests := map[string]string{
		"the same code":     "EUR",
		"lower case":        "eur",
		"surrounding space": " Eur ",
	}

	for name, code := range tests {
		euroID := s.InsertCurrency(s.T(), testhelpers.Euro())

		s.Run(name, func() {
			result := s.insertCurrency(ct.InsertCurrencyParams{Code: code, Name: "Euro", Symbol: "€"})

			require.Equal(s.T(), euroID, result.CurrencyID)
			require.Equal(s.T(), 1, s.countCurrencies())
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Currency)(nil), euroID)
	}
}
