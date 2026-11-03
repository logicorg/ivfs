package transactions

import (
	"errors"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestRecordExpense checks that an expense credits the balance it is paid
// from, an asset or a liability alike, debits its category, and is posted
// with the key prefixed by its type.
func (s *WorkflowUnitTestSuite) TestRecordExpense() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		input       tt.RecordExpenseInput
		description *string
	}{
		"paid by card, with a note": {
			input: tt.RecordExpenseInput{
				IdempotencyKey: "3f2a", BalanceLedgerAccountID: cardBalance.ID, CategoryLedgerAccountID: groceries.ID,
				Amount: decimal.RequireFromString("12.50"), OccurredAt: occurredAt, Description: "Esselunga",
			},
			description: ptr.NewPtr("Esselunga"),
		},
		"paid from cash, without a note": {
			input: tt.RecordExpenseInput{
				IdempotencyKey: "3f2a", BalanceLedgerAccountID: cash.ID, CategoryLedgerAccountID: groceries.ID,
				Amount: decimal.RequireFromString("12.50"), OccurredAt: occurredAt,
			},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			mockLedgerAccounts(env)
			env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, entry(lt.PostJournalEntryInput{
				IdempotencyKey: "expense:3f2a",
				OccurredAt:     occurredAt,
				Description:    tc.description,
				Postings: []lt.PostingInput{
					{LedgerAccountID: groceries.ID, Amount: decimal.RequireFromString("12.50"), Side: model.PostingDebit},
					{LedgerAccountID: tc.input.BalanceLedgerAccountID, Amount: decimal.RequireFromString("12.50"), Side: model.PostingCredit},
				},
			})).Return(&lt.PostJournalEntryResult{JournalEntryID: 41}, nil).Once()

			env.ExecuteWorkflow(RecordExpense, tc.input)

			require.NoError(s.T(), env.GetWorkflowError())
			var got *tt.RecordResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), &tt.RecordResult{JournalEntryID: 41}, got)
			env.AssertExpectations(s.T())
		})
	}
}

// TestRecordExpense_Errors checks that an invalid request, or accounts that
// cannot play their roles, fail before anything is posted, since
// PostJournalEntry is mocked only where a case reaches it, and that a failing
// child workflow's error is returned.
func (s *WorkflowUnitTestSuite) TestRecordExpense_Errors() {
	input := func(balance, category *model.LedgerAccount) tt.RecordExpenseInput {
		return tt.RecordExpenseInput{
			IdempotencyKey: "3f2a", BalanceLedgerAccountID: balance.ID, CategoryLedgerAccountID: category.ID,
			Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		input     tt.RecordExpenseInput
		mock      func(env *testsuite.TestWorkflowEnvironment)
		wantError string
	}{
		"an invalid request": {
			input:     tt.RecordExpenseInput{BalanceLedgerAccountID: cash.ID, CategoryLedgerAccountID: groceries.ID},
			wantError: "validation error",
		},
		"a balance that is a category": {
			input:     input(salary, groceries),
			wantError: `invalid parameter: "Salary" is not a balance`,
		},
		"a balance of no financial account": {
			input:     input(fxClearing, groceries),
			wantError: `invalid parameter: "FX clearing" is not a balance`,
		},
		"an income category": {
			input:     input(cash, salary),
			wantError: `invalid parameter: "Salary" is an income category, not an expense one`,
		},
		"a category that is a balance": {
			input:     input(cash, savingsVault),
			wantError: `invalid parameter: "Savings vault" is not an expense category: its kind is asset`,
		},
		"a category in another currency": {
			input:     input(cash, groceriesUSD),
			wantError: `invalid parameter: "Cash" and "Groceries (USD)" are in different currencies; record an exchange`,
		},
		"an unknown account": {
			input: tt.RecordExpenseInput{
				IdempotencyKey: "3f2a", BalanceLedgerAccountID: 999, CategoryLedgerAccountID: groceries.ID,
				Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
			},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: 999}).
					Return(nil, errors.New("resource LedgerAccount not found with ID: 999"))
			},
			wantError: "retrieve ledger account 999",
		},
		"a failed posting": {
			input: input(cash, groceries),
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, mock.Anything).
					Return(nil, errors.New("resource LedgerAccount with ID: 100 is closed"))
			},
			wantError: "resource LedgerAccount with ID: 100 is closed",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			if tc.mock != nil {
				tc.mock(env)
			}
			mockLedgerAccounts(env)

			env.ExecuteWorkflow(RecordExpense, tc.input)

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
