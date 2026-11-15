package transactions

import (
	"errors"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestRecordIncome checks that an income credits "Income" in the balance's
// currency with the category on that posting, debits the balance it arrives
// in, an asset or a liability alike, and is posted with the key prefixed by
// its type.
func (s *WorkflowUnitTestSuite) TestRecordIncome() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	amount := decimal.RequireFromString("2000.50")

	tests := []struct {
		name     string
		balance  *model.LedgerAccount
		category *model.Category
		// account is the shared account the category's side is booked on.
		account     *model.LedgerAccount
		description string
	}{
		{name: "a salary into cash, with a note", balance: cash, category: salary, account: income, description: "October salary"},
		{name: "a cashback onto the card, without a note", balance: cardBalance, category: salary, account: income},
		{name: "paid in dollars", balance: cashUSD, category: salary, account: incomeUSD},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			mockLedgerAccounts(env)
			mockCategories(env)
			want := lt.PostJournalEntryInput{
				IdempotencyKey: "income:77ab",
				OccurredAt:     occurredAt,
				Postings: []lt.PostingInput{
					{LedgerAccountID: tc.balance.ID, Amount: amount, Side: model.PostingDebit},
					{LedgerAccountID: tc.account.ID, Amount: amount, Side: model.PostingCredit, CategoryID: ptr.Int64(tc.category.ID)},
				},
			}
			if tc.description != "" {
				want.Description = ptr.NewPtr(tc.description)
			}
			env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, entry(want)).
				Return(&lt.PostJournalEntryResult{JournalEntryID: 43}, nil).Once()

			env.ExecuteWorkflow(RecordIncome, tt.RecordIncomeInput{
				IdempotencyKey: "77ab", BalanceLedgerAccountID: tc.balance.ID, CategoryID: tc.category.ID,
				Amount: amount, OccurredAt: occurredAt, Description: tc.description,
			})

			require.NoError(s.T(), env.GetWorkflowError())
			var got *tt.RecordResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), &tt.RecordResult{JournalEntryID: 43}, got)
			env.AssertExpectations(s.T())
		})
	}
}

// TestRecordIncome_Errors checks that an invalid request, or a balance or a
// category that cannot play its role, fails before anything is posted, since
// PostJournalEntry is mocked only where a case reaches it, and that a failing
// child workflow's error is returned.
func (s *WorkflowUnitTestSuite) TestRecordIncome_Errors() {
	tests := []struct {
		name string
		// key, balance, and category are what the request names.
		key               string
		balance, category int64
		// unknownAccount and unknownCategory make that retrieval fail.
		unknownAccount, unknownCategory int64
		// accountErr and postErr are what CreateLedgerAccount and
		// PostJournalEntry fail with, when reached.
		accountErr, postErr string
		wantError           string
	}{
		{name: "an invalid request", balance: cash.ID, category: salary.ID, wantError: "validation error"},
		{name: "a balance that is a shared account", key: "77ab", balance: expenses.ID, category: salary.ID, wantError: `invalid parameter: "Expenses" is not a balance`},
		{name: "a balance of no financial account", key: "77ab", balance: fxClearing.ID, category: salary.ID, wantError: `invalid parameter: "FX clearing" is not a balance`},
		{name: "a category of the other kind", key: "77ab", balance: cash.ID, category: groceries.ID, wantError: `invalid parameter: "Groceries" is an expense category, not an income one`},
		{name: "an unknown account", key: "77ab", balance: 999, category: salary.ID, unknownAccount: 999, wantError: "retrieve ledger account 999"},
		{name: "an unknown category", key: "77ab", balance: cash.ID, category: 999, unknownCategory: 999, wantError: "retrieve category 999"},
		{name: "a shared account that cannot be found", key: "77ab", balance: cash.ID, category: salary.ID, accountErr: "resource Currency with ID: 1 is closed", wantError: "find the Income account of currency 1"},
		{name: "a failed posting", key: "77ab", balance: cash.ID, category: salary.ID, postErr: "resource LedgerAccount with ID: 100 is closed", wantError: "resource LedgerAccount with ID: 100 is closed"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			if tc.unknownAccount != 0 {
				env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: tc.unknownAccount}).
					Return(nil, errors.New("resource LedgerAccount not found"))
			}
			if tc.unknownCategory != 0 {
				env.OnWorkflow(categorieswf.RetrieveCategory, mock.Anything, ct.RetrieveCategoryParams{ID: tc.unknownCategory}).
					Return(nil, errors.New("resource Category not found"))
			}
			if tc.accountErr != "" {
				env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, mock.Anything).Return(nil, errors.New(tc.accountErr))
			}
			if tc.postErr != "" {
				env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, mock.Anything).Return(nil, errors.New(tc.postErr))
			}
			mockLedgerAccounts(env)
			mockCategories(env)

			env.ExecuteWorkflow(RecordIncome, tt.RecordIncomeInput{
				IdempotencyKey: tc.key, BalanceLedgerAccountID: tc.balance, CategoryID: tc.category,
				Amount: decimal.RequireFromString("2000.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
			})

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
