package transactions

import (
	"errors"
	"time"

	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
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

// TestUpdateExpense checks that an update replaces the entry with the postings
// RecordExpense would post, crediting the balance and debiting "Expenses"
// with the category, and returns the warnings of the replace. Changing the
// category of an expense is this update: the posting stays on "Expenses".
func (s *WorkflowUnitTestSuite) TestUpdateExpense() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		balance     *model.LedgerAccount
		description string
		tags        []int64
		warnings    []lt.NegativeHolding
	}{
		{name: "paid by card, with a note and a tag", balance: cardBalance, description: "Esselunga", tags: []int64{5}},
		{name: "paid from cash, without a note", balance: cash},
		{name: "an entry that was a trade", balance: cash, warnings: soldSince},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			mockLedgerAccounts(env)
			mockCategories(env)
			want := jt.ReplaceJournalEntryInput{
				JournalEntryID: 41,
				OccurredAt:     occurredAt,
				Postings: []jt.PostingInput{
					{LedgerAccountID: expenses.ID, Amount: decimal.RequireFromString("12.50"), Side: model.PostingDebit, CategoryID: ptr.Int64(groceries.ID)},
					{LedgerAccountID: tc.balance.ID, Amount: decimal.RequireFromString("12.50"), Side: model.PostingCredit},
				},
			}
			if tc.description != "" {
				want.Description = ptr.NewPtr(tc.description)
			}
			want.TagIDs = tc.tags
			env.OnWorkflow(journalwf.ReplaceJournalEntry, mock.Anything, replacement(want)).
				Return(&jt.ReplaceJournalEntryResult{JournalEntryID: 41, Warnings: tc.warnings}, nil).Once()

			env.ExecuteWorkflow(UpdateExpense, tt.UpdateExpenseInput{
				JournalEntryID: 41, BalanceLedgerAccountID: tc.balance.ID, CategoryID: groceries.ID,
				Amount: decimal.RequireFromString("12.50"), OccurredAt: occurredAt, Description: tc.description,
				TagIDs: tc.tags,
			})

			require.NoError(s.T(), env.GetWorkflowError())
			var got *tt.UpdateResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), &tt.UpdateResult{JournalEntryID: 41, Warnings: tc.warnings}, got)
			env.AssertExpectations(s.T())
		})
	}
}

// TestUpdateExpense_Errors checks that an invalid request, or a balance or a
// category that cannot play its role, fails before anything is replaced, since
// ReplaceJournalEntry is mocked only where a case reaches it, and that a
// failing child workflow's error is returned.
func (s *WorkflowUnitTestSuite) TestUpdateExpense_Errors() {
	tests := []struct {
		name string
		// entry, balance, and category are the ids the request names.
		entry, balance, category int64
		// unknownAccount and unknownCategory make that retrieval fail.
		unknownAccount, unknownCategory int64
		// replaceErr is what ReplaceJournalEntry fails with, when reached.
		replaceErr string
		wantError  string
	}{
		{name: "an invalid request", balance: cash.ID, category: groceries.ID, wantError: "validation error"},
		{name: "a balance that is a shared account", entry: 41, balance: income.ID, category: groceries.ID, wantError: `invalid parameter: "Income" is not a balance`},
		{name: "a balance of no financial account", entry: 41, balance: fxClearing.ID, category: groceries.ID, wantError: `invalid parameter: "FX clearing" is not a balance`},
		{name: "a category of the other kind", entry: 41, balance: cash.ID, category: salary.ID, wantError: `invalid parameter: "Salary" is an income category, not an expense one`},
		{name: "an unknown category", entry: 41, balance: cash.ID, category: 999, unknownCategory: 999, wantError: "retrieve category 999"},
		{name: "an unknown account", entry: 41, balance: 999, category: groceries.ID, unknownAccount: 999, wantError: "retrieve ledger account 999"},
		{name: "an unknown entry", entry: 41, balance: cash.ID, category: groceries.ID, replaceErr: "resource JournalEntry not found with ID: 41", wantError: "update expense: "},
		{name: "a closed account the entry never touched", entry: 41, balance: cash.ID, category: groceries.ID, replaceErr: "resource LedgerAccount with ID: 100 is closed", wantError: "resource LedgerAccount with ID: 100 is closed"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			if tc.unknownAccount != 0 {
				env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, act.RetrieveLedgerAccountParams{ID: tc.unknownAccount}).
					Return(nil, errors.New("resource LedgerAccount not found"))
			}
			if tc.unknownCategory != 0 {
				env.OnWorkflow(categorieswf.RetrieveCategory, mock.Anything, ct.RetrieveCategoryParams{ID: tc.unknownCategory}).
					Return(nil, errors.New("resource Category not found"))
			}
			if tc.replaceErr != "" {
				env.OnWorkflow(journalwf.ReplaceJournalEntry, mock.Anything, mock.Anything).Return(nil, errors.New(tc.replaceErr))
			}
			mockLedgerAccounts(env)
			mockCategories(env)

			env.ExecuteWorkflow(UpdateExpense, tt.UpdateExpenseInput{
				JournalEntryID: tc.entry, BalanceLedgerAccountID: tc.balance, CategoryID: tc.category,
				Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
			})

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
