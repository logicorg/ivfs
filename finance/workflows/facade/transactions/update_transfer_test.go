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
)

// TestUpdateTransfer checks that an update replaces the entry with what
// RecordTransfer would post: two postings in one currency, and between
// currencies the postings and the metadata of an FX conversion without a fee,
// through the clearing account of each currency.
func (s *WorkflowUnitTestSuite) TestUpdateTransfer() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	amount, arrives := decimal.RequireFromString("300.50"), decimal.RequireFromString("324.15")

	tests := []struct {
		name     string
		input    tt.UpdateTransferInput
		want     lt.ReplaceJournalEntryInput
		warnings []lt.NegativeHolding
	}{
		{
			name: "paying the card off from cash, with a note",
			input: tt.UpdateTransferInput{
				JournalEntryID: 42, FromLedgerAccountID: cash.ID, ToLedgerAccountID: cardBalance.ID,
				Amount: amount, OccurredAt: occurredAt, Description: "pay the card off",
			},
			want: lt.ReplaceJournalEntryInput{
				JournalEntryID: 42, OccurredAt: occurredAt, Description: ptr.NewPtr("pay the card off"),
				Postings: []lt.PostingInput{
					{LedgerAccountID: cardBalance.ID, Amount: amount, Side: model.PostingDebit},
					{LedgerAccountID: cash.ID, Amount: amount, Side: model.PostingCredit},
				},
			},
		},
		{
			name: "into a savings vault, with the same amount arriving, on an entry that was a trade",
			input: tt.UpdateTransferInput{
				JournalEntryID: 42, FromLedgerAccountID: cash.ID, ToLedgerAccountID: savingsVault.ID,
				Amount: amount, ToAmount: decimal.RequireFromString("300.5"), OccurredAt: occurredAt,
			},
			want: lt.ReplaceJournalEntryInput{
				JournalEntryID: 42, OccurredAt: occurredAt,
				Postings: []lt.PostingInput{
					{LedgerAccountID: savingsVault.ID, Amount: amount, Side: model.PostingDebit},
					{LedgerAccountID: cash.ID, Amount: amount, Side: model.PostingCredit},
				},
			},
			warnings: soldSince,
		},
		{
			name: "from euros to dollars, with a note",
			input: tt.UpdateTransferInput{
				JournalEntryID: 42, FromLedgerAccountID: cash.ID, ToLedgerAccountID: cashUSD.ID,
				Amount: amount, ToAmount: arrives, OccurredAt: occurredAt, Description: "dollars for the trip",
			},
			want: lt.ReplaceJournalEntryInput{
				JournalEntryID: 42, OccurredAt: occurredAt, Description: ptr.NewPtr("dollars for the trip"),
				Metadata: map[string]string{"type": lt.FXConversionEntryType, "effective_rate": "324.15/300.5"},
				Postings: []lt.PostingInput{
					{LedgerAccountID: fxClearing.ID, Amount: amount, Side: model.PostingDebit},
					{LedgerAccountID: cash.ID, Amount: amount, Side: model.PostingCredit},
					{LedgerAccountID: cashUSD.ID, Amount: arrives, Side: model.PostingDebit},
					{LedgerAccountID: fxClearingUSD.ID, Amount: arrives, Side: model.PostingCredit},
				},
			},
		},
		{
			name: "from dollars to euros, without a note",
			input: tt.UpdateTransferInput{
				JournalEntryID: 42, FromLedgerAccountID: cashUSD.ID, ToLedgerAccountID: cash.ID,
				Amount: arrives, ToAmount: amount, OccurredAt: occurredAt,
			},
			want: lt.ReplaceJournalEntryInput{
				JournalEntryID: 42, OccurredAt: occurredAt,
				Metadata: map[string]string{"type": lt.FXConversionEntryType, "effective_rate": "300.5/324.15"},
				Postings: []lt.PostingInput{
					{LedgerAccountID: fxClearingUSD.ID, Amount: arrives, Side: model.PostingDebit},
					{LedgerAccountID: cashUSD.ID, Amount: arrives, Side: model.PostingCredit},
					{LedgerAccountID: cash.ID, Amount: amount, Side: model.PostingDebit},
					{LedgerAccountID: fxClearing.ID, Amount: amount, Side: model.PostingCredit},
				},
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			mockLedgerAccounts(env)
			mockFXClearing(env)
			env.OnWorkflow(journalwf.ReplaceJournalEntry, mock.Anything, replacement(tc.want)).
				Return(&lt.ReplaceJournalEntryResult{JournalEntryID: 42, Warnings: tc.warnings}, nil).Once()

			env.ExecuteWorkflow(UpdateTransfer, tc.input)

			require.NoError(s.T(), env.GetWorkflowError())
			var got *tt.UpdateResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), &tt.UpdateResult{JournalEntryID: 42, Warnings: tc.warnings}, got)
			env.AssertExpectations(s.T())
		})
	}
}

// TestUpdateTransfer_Errors checks that an invalid request, accounts that are
// not balances, or amounts that do not fit the balances' currencies fail
// before anything is replaced, since ReplaceJournalEntry is mocked only where
// a case reaches it, and that a failing child workflow's error is returned.
func (s *WorkflowUnitTestSuite) TestUpdateTransfer_Errors() {
	tests := []struct {
		name string
		// entry, from, and to are the ids the request names.
		entry, from, to int64
		toAmount        string
		// unknownAccount makes the retrieval of that account fail.
		unknownAccount int64
		// clearingErr is what finding an "FX clearing" account fails with.
		clearingErr string
		// replaceErr is what ReplaceJournalEntry fails with, when reached.
		replaceErr string
		wantError  string
	}{
		{name: "an invalid request", entry: 42, from: cash.ID, to: cash.ID, toAmount: "0", wantError: "invalid parameter: a transfer moves money between two different balances"},
		{name: "no entry", from: cash.ID, to: savingsVault.ID, toAmount: "0", wantError: "validation error"},
		{name: "a source that is a category", entry: 42, from: salary.ID, to: cash.ID, toAmount: "0", wantError: `invalid parameter: "Salary" is not a balance`},
		{name: "a destination of no financial account", entry: 42, from: cash.ID, to: fxClearing.ID, toAmount: "0", wantError: `invalid parameter: "FX clearing" is not a balance`},
		{name: "a different amount arriving in the same currency", entry: 42, from: cash.ID, to: savingsVault.ID, toAmount: "299", wantError: `invalid parameter: "Cash" and "Savings vault" are in the same currency`},
		{name: "no amount arriving in another currency", entry: 42, from: cash.ID, to: cashUSD.ID, toAmount: "0", wantError: `are in different currencies; give the amount that arrives as toAmount`},
		{name: "an unknown account", entry: 42, from: cash.ID, to: 999, toAmount: "0", unknownAccount: 999, wantError: "retrieve ledger account 999"},
		{name: "a failed clearing account lookup", entry: 42, from: cash.ID, to: cashUSD.ID, toAmount: "324.15", clearingErr: "resource Currency not found with ID: 1", wantError: "find the FX clearing account of currency 1"},
		{name: "an unknown entry", entry: 42, from: cash.ID, to: savingsVault.ID, toAmount: "0", replaceErr: "resource JournalEntry not found with ID: 42", wantError: "update transfer: "},
		{name: "a failed replace between currencies", entry: 42, from: cash.ID, to: cashUSD.ID, toAmount: "324.15", replaceErr: "resource LedgerAccount with ID: 103 is closed", wantError: "resource LedgerAccount with ID: 103 is closed"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			if tc.unknownAccount != 0 {
				env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: tc.unknownAccount}).
					Return(nil, errors.New("resource LedgerAccount not found"))
			}
			if tc.clearingErr != "" {
				env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, mock.Anything).Return(nil, errors.New(tc.clearingErr))
			}
			if tc.replaceErr != "" {
				env.OnWorkflow(journalwf.ReplaceJournalEntry, mock.Anything, mock.Anything).Return(nil, errors.New(tc.replaceErr))
			}
			mockLedgerAccounts(env)
			mockFXClearing(env)

			env.ExecuteWorkflow(UpdateTransfer, tt.UpdateTransferInput{
				JournalEntryID: tc.entry, FromLedgerAccountID: tc.from, ToLedgerAccountID: tc.to,
				Amount: decimal.NewFromInt(300), ToAmount: decimal.RequireFromString(tc.toAmount),
				OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
			})

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
