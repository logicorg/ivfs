package investments

import (
	"errors"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type WorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestWorkflowUnitTestSuite(t *testing.T) { suite.Run(t, new(WorkflowUnitTestSuite)) }

// ibkr is the brokerage account the trades are made in. It holds "Cash" and
// "Investments" in EUR, and only "Cash" in USD.
const ibkr int64 = 20

var (
	euro   = &model.Currency{ID: 1, Code: "EUR", Precision: 2}
	dollar = &model.Currency{ID: 2, Code: "USD", Precision: 2}

	cash        = model.LedgerAccount{ID: 500, FinancialAccountID: ptr.Int64(ibkr), CurrencyID: 1, Name: "Cash", Kind: model.LedgerAccountAsset}
	investments = model.LedgerAccount{ID: 501, FinancialAccountID: ptr.Int64(ibkr), CurrencyID: 1, Name: "investments", Kind: model.LedgerAccountAsset}
	cashUSD     = model.LedgerAccount{ID: 502, FinancialAccountID: ptr.Int64(ibkr), CurrencyID: 2, Name: "Cash", Kind: model.LedgerAccountAsset}

	// The categories a vest books to: in EUR, as ibkr's investments are, and
	// one in USD.
	compensation    = model.LedgerAccount{ID: 600, CurrencyID: 1, Name: "RSU compensation", Kind: model.LedgerAccountIncome}
	taxWithheld     = model.LedgerAccount{ID: 601, CurrencyID: 1, Name: "Tax withheld", Kind: model.LedgerAccountExpense}
	compensationUSD = model.LedgerAccount{ID: 602, CurrencyID: 2, Name: "RSU compensation (USD)", Kind: model.LedgerAccountIncome}
)

// unknownCategory is a ledger account ID no account has.
const unknownCategory int64 = 999

// mockCategories lets a vest retrieve its categories, or cash, by ID, and fail
// on unknownCategory as RetrieveLedgerAccount does. The mocks are optional, as
// a vest reads one category or two.
func mockCategories(env *testsuite.TestWorkflowEnvironment) {
	for _, account := range []model.LedgerAccount{compensation, taxWithheld, compensationUSD, cash} {
		env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: account.ID}).
			Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: &account}, nil).Maybe()
	}
	env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: unknownCategory}).
		Return(nil, errors.New("resource LedgerAccount not found with ID: 999")).Maybe()
}

// mockBrokerage lets the trades find the currencies, an unknown code failing
// as RetrieveCurrency does, and the balances of ibkr in each. The mocks are
// optional, as a case reads one currency or none.
func mockBrokerage(env *testsuite.TestWorkflowEnvironment) {
	for _, currency := range []*model.Currency{euro, dollar} {
		env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, &ct.RetrieveCurrencyParams{Code: currency.Code}).
			Return(currency, nil).Maybe()
	}
	env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, &ct.RetrieveCurrencyParams{Code: "XYZ"}).
		Return(nil, errors.New("resource Currency not found with code: XYZ")).Maybe()

	for currencyID, balances := range map[int64][]model.LedgerAccount{euro.ID: {cash, investments}, dollar.ID: {cashUSD}} {
		env.OnWorkflow(ledgeraccountswf.ListLedgerAccounts, mock.Anything, lt.ListLedgerAccountsParams{
			FinancialAccountID: ptr.Int64(ibkr),
			CurrencyID:         ptr.Int64(currencyID),
			Kinds:              []model.LedgerAccountKind{model.LedgerAccountAsset},
			IncludeClosed:      true,
		}).Return(&lt.ListLedgerAccountsResult{LedgerAccounts: balances}, nil).Maybe()
	}
}

// requireTrade compares trades by value, as tradeInput does, so a builder's
// 1053.00 equals the 1053 a case wants.
func requireTrade(t *testing.T, want, got trade) {
	t.Helper()
	require.True(t, want.amount.Equal(got.amount), "amount: want %s, got %s", want.amount, got.amount)
	require.True(t, want.unitsDelta.Equal(got.unitsDelta), "units: want %s, got %s", want.unitsDelta, got.unitsDelta)
	require.Equal(t, want.costBasis == nil, got.costBasis == nil, "whether the trade has a cost basis")
	if want.costBasis != nil {
		require.True(t, want.costBasis.Equal(*got.costBasis), "cost basis: want %s, got %s", want.costBasis, got.costBasis)
	}
	want.amount, want.unitsDelta, want.costBasis = got.amount, got.unitsDelta, got.costBasis
	require.Equal(t, want, got)
}

// normalizeVest rewrites the decimals of a vest from their strings, so two
// vests recording the same values compare equal whatever their scale: 600.00
// and 600 both become 600.
func normalizeVest(v lt.RegisterVestInput) lt.RegisterVestInput {
	normalize := func(d decimal.Decimal) decimal.Decimal { return decimal.RequireFromString(d.String()) }
	v.Units, v.CostBasisAmount = normalize(v.Units), normalize(v.CostBasisAmount)
	postings := make([]lt.PostingInput, len(v.Entry.Postings))
	for i, posting := range v.Entry.Postings {
		posting.Amount = normalize(posting.Amount)
		postings[i] = posting
	}
	v.Entry.Postings = postings
	return v
}

// tradeInput matches the trade want by what it records rather than by its
// representation: a decimal such as 1053.00 arrives as 1053, equal in value
// but not in its fields.
func tradeInput(want lt.RegisterInvestmentTradeInput) any {
	sameDecimal := func(a, b *decimal.Decimal) bool { return (a == nil) == (b == nil) && (a == nil || a.Equal(*b)) }
	sameID := func(a, b *int64) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	sameString := func(a, b *string) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

	return mock.MatchedBy(func(got lt.RegisterInvestmentTradeInput) bool {
		if got.FinancialAccountID != want.FinancialAccountID || got.AssetID != want.AssetID ||
			!got.UnitsDelta.Equal(want.UnitsDelta) || !sameDecimal(got.CostBasisAmount, want.CostBasisAmount) ||
			!sameID(got.CostBasisCurrencyID, want.CostBasisCurrencyID) {
			return false
		}
		gotEntry, wantEntry := got.Entry, want.Entry
		if gotEntry.IdempotencyKey != wantEntry.IdempotencyKey || !gotEntry.OccurredAt.Equal(wantEntry.OccurredAt) ||
			!sameString(gotEntry.Description, wantEntry.Description) || gotEntry.ReversalOfEntryID != nil ||
			gotEntry.CounterpartyID != nil || len(gotEntry.Metadata) != 0 || len(gotEntry.Postings) != len(wantEntry.Postings) {
			return false
		}
		for i, posting := range wantEntry.Postings {
			if gotEntry.Postings[i].LedgerAccountID != posting.LedgerAccountID || gotEntry.Postings[i].Side != posting.Side ||
				!gotEntry.Postings[i].Amount.Equal(posting.Amount) {
				return false
			}
		}
		return true
	})
}
