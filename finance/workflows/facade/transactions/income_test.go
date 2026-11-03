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

// TestRecordIncome checks that an income credits its category, debits the
// balance it arrives in, an asset or a liability alike, and is posted with the
// key prefixed by its type.
func (s *WorkflowUnitTestSuite) TestRecordIncome() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		input       tt.RecordIncomeInput
		description *string
	}{
		"a salary into cash, with a note": {
			input: tt.RecordIncomeInput{
				IdempotencyKey: "77ab", BalanceLedgerAccountID: cash.ID, CategoryLedgerAccountID: salary.ID,
				Amount: decimal.RequireFromString("2000.50"), OccurredAt: occurredAt, Description: "October salary",
			},
			description: ptr.NewPtr("October salary"),
		},
		"a cashback onto the card, without a note": {
			input: tt.RecordIncomeInput{
				IdempotencyKey: "77ab", BalanceLedgerAccountID: cardBalance.ID, CategoryLedgerAccountID: salary.ID,
				Amount: decimal.RequireFromString("2000.50"), OccurredAt: occurredAt,
			},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			mockLedgerAccounts(env)
			env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, entry(lt.PostJournalEntryInput{
				IdempotencyKey: "income:77ab",
				OccurredAt:     occurredAt,
				Description:    tc.description,
				Postings: []lt.PostingInput{
					{LedgerAccountID: tc.input.BalanceLedgerAccountID, Amount: decimal.RequireFromString("2000.50"), Side: model.PostingDebit},
					{LedgerAccountID: salary.ID, Amount: decimal.RequireFromString("2000.50"), Side: model.PostingCredit},
				},
			})).Return(&lt.PostJournalEntryResult{JournalEntryID: 43}, nil).Once()

			env.ExecuteWorkflow(RecordIncome, tc.input)

			require.NoError(s.T(), env.GetWorkflowError())
			var got *tt.RecordResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), &tt.RecordResult{JournalEntryID: 43}, got)
			env.AssertExpectations(s.T())
		})
	}
}

// TestRecordIncome_Errors checks that an invalid request, or accounts that
// cannot play their roles, fail before anything is posted, since
// PostJournalEntry is mocked only where a case reaches it, and that a failing
// child workflow's error is returned.
func (s *WorkflowUnitTestSuite) TestRecordIncome_Errors() {
	input := func(balance, category *model.LedgerAccount) tt.RecordIncomeInput {
		return tt.RecordIncomeInput{
			IdempotencyKey: "77ab", BalanceLedgerAccountID: balance.ID, CategoryLedgerAccountID: category.ID,
			Amount: decimal.NewFromInt(2000), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		input     tt.RecordIncomeInput
		mock      func(env *testsuite.TestWorkflowEnvironment)
		wantError string
	}{
		"an invalid request": {
			input:     tt.RecordIncomeInput{BalanceLedgerAccountID: cash.ID, CategoryLedgerAccountID: salary.ID},
			wantError: "validation error",
		},
		"a balance that is a category": {
			input:     input(groceries, salary),
			wantError: `invalid parameter: "Groceries" is not a balance`,
		},
		"a balance of no financial account": {
			input:     input(openingBalance, salary),
			wantError: `invalid parameter: "Opening balances" is not a balance`,
		},
		"an expense category": {
			input:     input(cash, groceries),
			wantError: `invalid parameter: "Groceries" is an expense category, not an income one`,
		},
		"a category that is a balance": {
			input:     input(cash, cardBalance),
			wantError: `invalid parameter: "Card balance" is not an income category: its kind is liability`,
		},
		"a category in another currency": {
			input:     input(cash, salaryUSD),
			wantError: `invalid parameter: "Cash" and "Salary (USD)" are in different currencies; record an exchange`,
		},
		"an unknown account": {
			input: tt.RecordIncomeInput{
				IdempotencyKey: "77ab", BalanceLedgerAccountID: cash.ID, CategoryLedgerAccountID: 999,
				Amount: decimal.NewFromInt(2000), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
			},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: 999}).
					Return(nil, errors.New("resource LedgerAccount not found with ID: 999"))
			},
			wantError: "retrieve ledger account 999",
		},
		"a failed posting": {
			input: input(cash, salary),
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, mock.Anything).
					Return(nil, errors.New("resource LedgerAccount with ID: 210 is closed"))
			},
			wantError: "resource LedgerAccount with ID: 210 is closed",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			if tc.mock != nil {
				tc.mock(env)
			}
			mockLedgerAccounts(env)

			env.ExecuteWorkflow(RecordIncome, tc.input)

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
