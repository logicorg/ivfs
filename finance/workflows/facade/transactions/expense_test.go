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

// TestRecordExpense checks that an expense credits the balance it is paid
// from, an asset or a liability alike, debits "Expenses" in the balance's
// currency with the category on that posting, and is posted with the key
// prefixed by its type.
func (s *WorkflowUnitTestSuite) TestRecordExpense() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	amount := decimal.RequireFromString("12.50")

	tests := []struct {
		name     string
		balance  *model.LedgerAccount
		category *model.Category
		// account is the shared account the category's side is booked on.
		account     *model.LedgerAccount
		description string
		tags        []int64
	}{
		{name: "paid by card, with a note and a tag", balance: cardBalance, category: groceries, account: expenses, description: "Esselunga", tags: []int64{5}},
		{name: "paid from cash, without a note", balance: cash, category: groceries, account: expenses},
		{name: "paid in dollars", balance: cashUSD, category: groceries, account: expensesUSD},
		{name: "for a closed category", balance: cash, category: oldHobby, account: expenses},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			mockLedgerAccounts(env)
			mockCategories(env)
			want := lt.PostJournalEntryInput{
				IdempotencyKey: "expense:3f2a",
				OccurredAt:     occurredAt,
				Postings: []lt.PostingInput{
					{LedgerAccountID: tc.account.ID, Amount: amount, Side: model.PostingDebit, CategoryID: ptr.Int64(tc.category.ID)},
					{LedgerAccountID: tc.balance.ID, Amount: amount, Side: model.PostingCredit},
				},
			}
			if tc.description != "" {
				want.Description = ptr.NewPtr(tc.description)
			}
			want.TagIDs = tc.tags
			env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, entry(want)).
				Return(&lt.PostJournalEntryResult{JournalEntryID: 41}, nil).Once()

			env.ExecuteWorkflow(RecordExpense, tt.RecordExpenseInput{
				IdempotencyKey: "3f2a", BalanceLedgerAccountID: tc.balance.ID, CategoryID: tc.category.ID,
				Amount: amount, OccurredAt: occurredAt, Description: tc.description,
				TagIDs: tc.tags,
			})

			require.NoError(s.T(), env.GetWorkflowError())
			var got *tt.RecordResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), &tt.RecordResult{JournalEntryID: 41}, got)
			env.AssertExpectations(s.T())
		})
	}
}

// TestRecordExpense_Errors checks that an invalid request, or a balance or a
// category that cannot play its role, fails before anything is posted, since
// PostJournalEntry is mocked only where a case reaches it, and that a failing
// child workflow's error is returned.
func (s *WorkflowUnitTestSuite) TestRecordExpense_Errors() {
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
		{name: "an invalid request", balance: cash.ID, category: groceries.ID, wantError: "validation error"},
		{name: "a balance that is a shared account", key: "3f2a", balance: income.ID, category: groceries.ID, wantError: `invalid parameter: "Income" is not a balance`},
		{name: "a balance of no financial account", key: "3f2a", balance: fxClearing.ID, category: groceries.ID, wantError: `invalid parameter: "FX clearing" is not a balance`},
		{name: "a category of the other kind", key: "3f2a", balance: cash.ID, category: salary.ID, wantError: `invalid parameter: "Salary" is an income category, not an expense one`},
		{name: "an unknown account", key: "3f2a", balance: 999, category: groceries.ID, unknownAccount: 999, wantError: "retrieve ledger account 999"},
		{name: "an unknown category", key: "3f2a", balance: cash.ID, category: 999, unknownCategory: 999, wantError: "retrieve category 999"},
		{name: "a shared account that cannot be found", key: "3f2a", balance: cash.ID, category: groceries.ID, accountErr: "resource Currency with ID: 1 is closed", wantError: "find the Expenses account of currency 1"},
		{name: "a failed posting", key: "3f2a", balance: cash.ID, category: groceries.ID, postErr: "resource LedgerAccount with ID: 100 is closed", wantError: "resource LedgerAccount with ID: 100 is closed"},
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

			env.ExecuteWorkflow(RecordExpense, tt.RecordExpenseInput{
				IdempotencyKey: tc.key, BalanceLedgerAccountID: tc.balance, CategoryID: tc.category,
				Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
			})

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
