package accounts

import (
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// balanceFixture is a ledger account of each balance closing cares about.
type balanceFixture struct {
	// cash is an asset holding 100.
	cash *model.LedgerAccount
	// emptied is an asset that held 50 and was emptied.
	emptied *model.LedgerAccount
	// drafted is an asset with only a draft entry of 70.
	drafted *model.LedgerAccount
	// card is a liability owing 30.
	card *model.LedgerAccount
	// salary is income of 150.
	salary *model.LedgerAccount
	// unused is an asset with no entries.
	unused *model.LedgerAccount
}

// insertBalanceFixture inserts the accounts of a balanceFixture and their
// entries. A posted entry cannot be deleted, so the fixture is inserted once
// per test, and each case closes an account of its own.
func insertBalanceFixture(t *testing.T, s *UnitTestSuite) balanceFixture {
	t.Helper()

	currency := testhelpers.Euro()
	temporaltesting.Insert(t, s.DB, currency)
	account := func(name string, kind model.LedgerAccountKind) *model.LedgerAccount {
		ledgerAccount := &model.LedgerAccount{Name: name, CurrencyID: currency.ID, Kind: kind}
		temporaltesting.Insert(t, s.DB, ledgerAccount)
		return ledgerAccount
	}
	f := balanceFixture{
		cash:    account("Cash", model.LedgerAccountAsset),
		emptied: account("Emptied", model.LedgerAccountAsset),
		drafted: account("Drafted", model.LedgerAccountAsset),
		card:    account("Card", model.LedgerAccountLiability),
		salary:  account("Salary", model.LedgerAccountIncome),
		unused:  account("Unused", model.LedgerAccountAsset),
	}
	groceries := account("Groceries", model.LedgerAccountExpense)

	occurredAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	post := func(debit, credit *model.LedgerAccount, amount int64) {
		temporaltesting.InsertPostedJournalEntry(t, s.DB, &model.JournalEntry{OccurredAt: occurredAt},
			model.Posting{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingDebit},
			model.Posting{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingCredit},
		)
	}
	post(f.cash, f.salary, 100)
	post(f.emptied, f.salary, 50)
	post(groceries, f.emptied, 50)
	post(groceries, f.card, 30)

	draft := &model.JournalEntry{OccurredAt: occurredAt, Status: model.JournalEntryDraft}
	temporaltesting.Insert(t, s.DB, draft)
	temporaltesting.Insert(t, s.DB, &model.Posting{
		JournalEntryID: draft.ID, LineNumber: 1, LedgerAccountID: f.drafted.ID, Amount: decimal.NewFromInt(70), Side: model.PostingDebit,
	})
	return f
}

// TestCloseLedgerAccountRefusesStrandedBalance checks that an asset or
// liability closes only without a balance, counting posted entries only, while
// income closes with one, and that a refused close leaves the account open.
func (s *UnitTestSuite) TestCloseLedgerAccountRefusesStrandedBalance() {
	f := insertBalanceFixture(s.T(), s)
	closedAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		account *model.LedgerAccount
		errStr  string
	}{
		"an asset with a balance": {
			account: f.cash,
			errStr:  "resource LedgerAccount conflict: ledger account still has a balance of 100",
		},
		"a liability with a balance, in its natural direction": {
			account: f.card,
			errStr:  "resource LedgerAccount conflict: ledger account still has a balance of 30",
		},
		"an asset that was emptied":         {account: f.emptied},
		"an asset with only a draft entry":  {account: f.drafted},
		"an asset with no entries":          {account: f.unused},
		"income with a balance, as history": {account: f.salary},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.CloseLedgerAccount, lt.CloseLedgerAccountInput{LedgerAccountID: tc.account.ID, ClosedAt: closedAt})

			var stored model.LedgerAccount
			temporaltesting.Retrieve(s.T(), s.DB, &stored, tc.account.ID)
			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
				require.Nil(s.T(), stored.ClosedAt)
				return
			}
			require.NoError(s.T(), err)
			require.NotNil(s.T(), stored.ClosedAt)
			require.Equal(s.T(), closedAt, stored.ClosedAt.UTC())
		})
	}
}
