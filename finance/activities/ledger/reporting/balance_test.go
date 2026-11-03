package reporting

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Test_LedgerAccountBalance checks that a balance is reported in the account's
// natural direction, cut off at the requested instant, and together with the
// account's name, kind, and currency code, and whether it is closed. The
// entries are seeded once and only read by the cases, since a posted entry
// cannot be deleted.
func (s *UnitTestSuite) Test_LedgerAccountBalance() {
	fixture := s.buildReportingFixture(s.T())

	cash := &model.LedgerAccount{FinancialAccountID: &fixture.BankFinancialAccount.ID, CurrencyID: fixture.Currency.ID, Name: "Revolut EUR", Kind: model.LedgerAccountAsset}
	card := &model.LedgerAccount{CurrencyID: fixture.Currency.ID, Name: "Credit card", Kind: model.LedgerAccountLiability}
	salary := &model.LedgerAccount{CurrencyID: fixture.Currency.ID, Name: "Salary", Kind: model.LedgerAccountIncome}
	groceries := &model.LedgerAccount{CurrencyID: fixture.Currency.ID, Name: "Groceries", Kind: model.LedgerAccountExpense}
	closedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	gifts := &model.LedgerAccount{CurrencyID: fixture.Currency.ID, Name: "Gifts", Kind: model.LedgerAccountExpense, ClosedAt: &closedAt}
	for _, account := range []*model.LedgerAccount{cash, card, salary, groceries, gifts} {
		temporaltesting.Insert(s.T(), s.DB, account)
	}

	january := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	february := january.AddDate(0, 1, 0)
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, &model.JournalEntry{OccurredAt: january},
		model.Posting{LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(1000), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: salary.ID, Amount: decimal.NewFromInt(1000), Side: model.PostingCredit},
	)
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, &model.JournalEntry{OccurredAt: february},
		model.Posting{LedgerAccountID: groceries.ID, Amount: decimal.NewFromInt(30), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: card.ID, Amount: decimal.NewFromInt(30), Side: model.PostingCredit},
	)

	tests := map[string]struct {
		account     func() *model.LedgerAccount
		at          time.Time
		wantBalance int64
	}{
		"an asset, debit-normal":     {account: func() *model.LedgerAccount { return cash }, wantBalance: 1000},
		"a liability, credit-normal": {account: func() *model.LedgerAccount { return card }, wantBalance: 30},
		"income, credit-normal":      {account: func() *model.LedgerAccount { return salary }, wantBalance: 1000},
		"an expense, debit-normal":   {account: func() *model.LedgerAccount { return groceries }, wantBalance: 30},
		"before its first entry":     {account: func() *model.LedgerAccount { return groceries }, at: january, wantBalance: 0},
		"at the instant of an entry": {account: func() *model.LedgerAccount { return cash }, at: january, wantBalance: 1000},
		"a closed account":           {account: func() *model.LedgerAccount { return gifts }, wantBalance: 0},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			account := tc.account()
			activityResult, err := s.Env.ExecuteActivity(s.ra.LedgerAccountBalance, lt.LedgerAccountBalanceParams{LedgerAccountID: account.ID, At: tc.at})
			require.NoError(s.T(), err)

			var got lt.LedgerAccountBalanceResult
			require.NoError(s.T(), activityResult.Get(&got))
			require.Truef(s.T(), got.Balance.Equal(decimal.NewFromInt(tc.wantBalance)), "balance %s, want %d", got.Balance, tc.wantBalance)
			got.Balance = decimal.Decimal{}
			require.Equal(s.T(), lt.LedgerAccountBalanceResult{
				LedgerAccountID:   account.ID,
				LedgerAccountName: account.Name,
				Kind:              account.Kind,
				CurrencyID:        fixture.Currency.ID,
				CurrencyCode:      "EUR",
				Closed:            account.ClosedAt != nil,
			}, got)
		})
	}
}

func (s *UnitTestSuite) Test_LedgerAccountBalance_Errors() {
	tests := map[string]struct {
		ledgerAccountID int64
		wantError       string
	}{
		"no id":         {ledgerAccountID: 0, wantError: "invalid parameter"},
		"a negative id": {ledgerAccountID: -1, wantError: "invalid parameter"},
		"an unknown id": {ledgerAccountID: 999999, wantError: "resource LedgerAccount not found"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.ra.LedgerAccountBalance, lt.LedgerAccountBalanceParams{LedgerAccountID: tc.ledgerAccountID})
			require.ErrorContains(s.T(), err, tc.wantError)
		})
	}
}
