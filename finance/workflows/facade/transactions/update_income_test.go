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

// TestUpdateIncome checks that an update replaces the entry with the postings
// RecordIncome would post, crediting "Income" with the category and debiting
// the balance, and returns the warnings of the replace.
func (s *WorkflowUnitTestSuite) TestUpdateIncome() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		balance     *model.LedgerAccount
		description string
		tags        []int64
		warnings    []lt.NegativeHolding
	}{
		{name: "a cashback onto the card, with a note and a tag", balance: cardBalance, description: "October salary", tags: []int64{5}},
		{name: "a salary into cash, without a note", balance: cash},
		{name: "an entry that was a trade", balance: cash, warnings: soldSince},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			mockLedgerAccounts(env)
			mockCategories(env)
			want := lt.ReplaceJournalEntryInput{
				JournalEntryID: 41,
				OccurredAt:     occurredAt,
				Postings: []lt.PostingInput{
					{LedgerAccountID: tc.balance.ID, Amount: decimal.RequireFromString("12.50"), Side: model.PostingDebit},
					{LedgerAccountID: income.ID, Amount: decimal.RequireFromString("12.50"), Side: model.PostingCredit, CategoryID: ptr.Int64(salary.ID)},
				},
			}
			if tc.description != "" {
				want.Description = ptr.NewPtr(tc.description)
			}
			want.TagIDs = tc.tags
			env.OnWorkflow(journalwf.ReplaceJournalEntry, mock.Anything, replacement(want)).
				Return(&lt.ReplaceJournalEntryResult{JournalEntryID: 41, Warnings: tc.warnings}, nil).Once()

			env.ExecuteWorkflow(UpdateIncome, tt.UpdateIncomeInput{
				JournalEntryID: 41, BalanceLedgerAccountID: tc.balance.ID, CategoryID: salary.ID,
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

// TestUpdateIncome_Errors checks that an invalid request, or a balance or a
// category that cannot play its role, fails before anything is replaced, since
// ReplaceJournalEntry is mocked only where a case reaches it, and that a
// failing child workflow's error is returned.
func (s *WorkflowUnitTestSuite) TestUpdateIncome_Errors() {
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
		{name: "an invalid request", balance: cash.ID, category: salary.ID, wantError: "validation error"},
		{name: "a balance that is a shared account", entry: 41, balance: expenses.ID, category: salary.ID, wantError: `invalid parameter: "Expenses" is not a balance`},
		{name: "a balance of no financial account", entry: 41, balance: fxClearing.ID, category: salary.ID, wantError: `invalid parameter: "FX clearing" is not a balance`},
		{name: "a category of the other kind", entry: 41, balance: cash.ID, category: groceries.ID, wantError: `invalid parameter: "Groceries" is an expense category, not an income one`},
		{name: "an unknown category", entry: 41, balance: cash.ID, category: 999, unknownCategory: 999, wantError: "retrieve category 999"},
		{name: "an unknown account", entry: 41, balance: 999, category: salary.ID, unknownAccount: 999, wantError: "retrieve ledger account 999"},
		{name: "an unknown entry", entry: 41, balance: cash.ID, category: salary.ID, replaceErr: "resource JournalEntry not found with ID: 41", wantError: "update income: "},
		{name: "a closed account the entry never touched", entry: 41, balance: cash.ID, category: salary.ID, replaceErr: "resource LedgerAccount with ID: 100 is closed", wantError: "resource LedgerAccount with ID: 100 is closed"},
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
			if tc.replaceErr != "" {
				env.OnWorkflow(journalwf.ReplaceJournalEntry, mock.Anything, mock.Anything).Return(nil, errors.New(tc.replaceErr))
			}
			mockLedgerAccounts(env)
			mockCategories(env)

			env.ExecuteWorkflow(UpdateIncome, tt.UpdateIncomeInput{
				JournalEntryID: tc.entry, BalanceLedgerAccountID: tc.balance, CategoryID: tc.category,
				Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
			})

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
