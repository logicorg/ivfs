package transactions

import (
	"errors"
	"slices"
	"time"

	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	fxt "github.com/logicorg/ivfs/finance/common/types/fxtypes"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	ledgerfxwf "github.com/logicorg/ivfs/finance/workflows/ledger/fx"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// mockFXClearing lets CreateLedgerAccount find the "FX clearing" account of
// each fixture currency.
func mockFXClearing(env *testsuite.TestWorkflowEnvironment) {
	for _, account := range []*model.LedgerAccount{fxClearing, fxClearingUSD} {
		env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, act.CreateLedgerAccountInput{
			CurrencyID: account.CurrencyID, Name: "FX clearing", Kind: model.LedgerAccountAsset,
		}).Return(&act.CreateLedgerAccountResult{LedgerAccountID: account.ID}, nil).Maybe()
	}
}

// conversion matches the FX conversion want by what it books rather than by
// its representation, as entry does for a journal entry.
func conversion(want fxt.RegisterFXConversionInput) any {
	return mock.MatchedBy(func(got fxt.RegisterFXConversionInput) bool {
		sameDescription := (got.Description == nil) == (want.Description == nil) &&
			(got.Description == nil || *got.Description == *want.Description)
		sameFee := (got.Fee == nil) == (want.Fee == nil) &&
			(got.Fee == nil || got.Fee.CategoryID == want.Fee.CategoryID && got.Fee.Amount.Equal(want.Fee.Amount) &&
				got.Fee.LedgerAccountID == want.Fee.LedgerAccountID)
		return got.IdempotencyKey == want.IdempotencyKey && got.OccurredAt.Equal(want.OccurredAt) && sameDescription &&
			got.CounterpartyID == nil && got.ExternalID == nil && got.ProviderReference == nil && sameFee &&
			got.SourceLedgerAccountID == want.SourceLedgerAccountID &&
			got.SourceClearingLedgerAccountID == want.SourceClearingLedgerAccountID &&
			got.SourceAmount.Equal(want.SourceAmount) &&
			got.DestinationLedgerAccountID == want.DestinationLedgerAccountID &&
			got.DestinationClearingLedgerAccountID == want.DestinationClearingLedgerAccountID &&
			got.DestinationAmount.Equal(want.DestinationAmount) &&
			slices.Equal(got.TagIDs, want.TagIDs)
	})
}

// TestRecordTransfer checks that a transfer in one currency credits the
// balance the money leaves and debits the one it reaches, whatever their
// kinds, with each charge debited on "Expenses" out of what it reaches, and
// that a transfer between currencies is posted as an FX conversion through
// the clearing account of each currency, its charge as the fee, both with the
// key prefixed by the transaction's type.
func (s *WorkflowUnitTestSuite) TestRecordTransfer() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		input tt.RecordTransferInput
		// mock mocks the child workflow that posts, returning entry 42.
		mock func(env *testsuite.TestWorkflowEnvironment)
	}{
		"paying the card off from cash, with a note and a tag": {
			input: tt.RecordTransferInput{
				IdempotencyKey: "9c1e", FromLedgerAccountID: cash.ID, ToLedgerAccountID: cardBalance.ID,
				Amount: decimal.RequireFromString("300.50"), OccurredAt: occurredAt, Description: "pay the card off",
				TagIDs: []int64{5},
			},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, entry(jt.PostJournalEntryInput{
					IdempotencyKey: "transfer:9c1e",
					OccurredAt:     occurredAt,
					Description:    ptr.NewPtr("pay the card off"),
					TagIDs:         []int64{5},
					Postings: []jt.PostingInput{
						{LedgerAccountID: cardBalance.ID, Amount: decimal.RequireFromString("300.50"), Side: model.PostingDebit},
						{LedgerAccountID: cash.ID, Amount: decimal.RequireFromString("300.50"), Side: model.PostingCredit},
					},
				})).Return(&jt.PostJournalEntryResult{JournalEntryID: 42}, nil).Once()
			},
		},
		"into a savings vault, with the same amount arriving": {
			input: tt.RecordTransferInput{
				IdempotencyKey: "9c1e", FromLedgerAccountID: cash.ID, ToLedgerAccountID: savingsVault.ID,
				Amount: decimal.RequireFromString("300.50"), ToAmount: decimal.RequireFromString("300.5"), OccurredAt: occurredAt,
			},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, entry(jt.PostJournalEntryInput{
					IdempotencyKey: "transfer:9c1e",
					OccurredAt:     occurredAt,
					Postings: []jt.PostingInput{
						{LedgerAccountID: savingsVault.ID, Amount: decimal.RequireFromString("300.50"), Side: model.PostingDebit},
						{LedgerAccountID: cash.ID, Amount: decimal.RequireFromString("300.50"), Side: model.PostingCredit},
					},
				})).Return(&jt.PostJournalEntryResult{JournalEntryID: 42}, nil).Once()
			},
		},
		"from euros to dollars, with a note and a tag": {
			input: tt.RecordTransferInput{
				IdempotencyKey: "9c1e", FromLedgerAccountID: cash.ID, ToLedgerAccountID: cashUSD.ID,
				Amount: decimal.RequireFromString("300.50"), ToAmount: decimal.RequireFromString("324.15"), OccurredAt: occurredAt,
				Description: "dollars for the trip", TagIDs: []int64{5},
			},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(ledgerfxwf.RegisterFXConversion, mock.Anything, conversion(fxt.RegisterFXConversionInput{
					IdempotencyKey:                     "transfer:9c1e",
					OccurredAt:                         occurredAt,
					Description:                        ptr.NewPtr("dollars for the trip"),
					TagIDs:                             []int64{5},
					SourceLedgerAccountID:              cash.ID,
					SourceClearingLedgerAccountID:      fxClearing.ID,
					SourceAmount:                       decimal.RequireFromString("300.50"),
					DestinationLedgerAccountID:         cashUSD.ID,
					DestinationClearingLedgerAccountID: fxClearingUSD.ID,
					DestinationAmount:                  decimal.RequireFromString("324.15"),
				})).Return(&fxt.RegisterFXConversionResult{JournalEntryID: 42}, nil).Once()
			},
		},
		"a mortgage instalment, part of it interest": {
			input: tt.RecordTransferInput{
				IdempotencyKey: "9c1e", FromLedgerAccountID: cash.ID, ToLedgerAccountID: principal.ID,
				Amount: decimal.RequireFromString("1200.00"), OccurredAt: occurredAt,
				Charges: []tt.Split{{CategoryID: interest.ID, Amount: decimal.RequireFromString("800.00")}},
			},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, entry(jt.PostJournalEntryInput{
					IdempotencyKey: "transfer:9c1e",
					OccurredAt:     occurredAt,
					Postings: []jt.PostingInput{
						{LedgerAccountID: principal.ID, Amount: decimal.NewFromInt(400), Side: model.PostingDebit},
						{LedgerAccountID: expenses.ID, Amount: decimal.NewFromInt(800), Side: model.PostingDebit, CategoryID: ptr.Int64(interest.ID)},
						{LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(1200), Side: model.PostingCredit},
					},
				})).Return(&jt.PostJournalEntryResult{JournalEntryID: 42}, nil).Once()
			},
		},
		"two charges, with what arrives given": {
			input: tt.RecordTransferInput{
				IdempotencyKey: "9c1e", FromLedgerAccountID: cash.ID, ToLedgerAccountID: principal.ID,
				Amount: decimal.NewFromInt(1200), ToAmount: decimal.RequireFromString("350.50"), OccurredAt: occurredAt,
				Charges: []tt.Split{
					{CategoryID: interest.ID, Amount: decimal.NewFromInt(800)},
					{CategoryID: groceries.ID, Amount: decimal.RequireFromString("49.50")},
				},
			},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, entry(jt.PostJournalEntryInput{
					IdempotencyKey: "transfer:9c1e",
					OccurredAt:     occurredAt,
					Postings: []jt.PostingInput{
						{LedgerAccountID: principal.ID, Amount: decimal.RequireFromString("350.50"), Side: model.PostingDebit},
						{LedgerAccountID: expenses.ID, Amount: decimal.NewFromInt(800), Side: model.PostingDebit, CategoryID: ptr.Int64(interest.ID)},
						{LedgerAccountID: expenses.ID, Amount: decimal.RequireFromString("49.50"), Side: model.PostingDebit, CategoryID: ptr.Int64(groceries.ID)},
						{LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(1200), Side: model.PostingCredit},
					},
				})).Return(&jt.PostJournalEntryResult{JournalEntryID: 42}, nil).Once()
			},
		},
		"from euros to dollars, with a charge as the fee": {
			input: tt.RecordTransferInput{
				IdempotencyKey: "9c1e", FromLedgerAccountID: cash.ID, ToLedgerAccountID: cashUSD.ID,
				Amount: decimal.RequireFromString("300.50"), ToAmount: decimal.RequireFromString("324.15"), OccurredAt: occurredAt,
				Charges: []tt.Split{{CategoryID: groceries.ID, Amount: decimal.RequireFromString("1.50")}},
			},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(ledgerfxwf.RegisterFXConversion, mock.Anything, conversion(fxt.RegisterFXConversionInput{
					IdempotencyKey:                     "transfer:9c1e",
					OccurredAt:                         occurredAt,
					SourceLedgerAccountID:              cash.ID,
					SourceClearingLedgerAccountID:      fxClearing.ID,
					SourceAmount:                       decimal.RequireFromString("300.50"),
					DestinationLedgerAccountID:         cashUSD.ID,
					DestinationClearingLedgerAccountID: fxClearingUSD.ID,
					DestinationAmount:                  decimal.RequireFromString("324.15"),
					Fee:                                &fxt.FXFee{CategoryID: groceries.ID, Amount: decimal.RequireFromString("1.50"), LedgerAccountID: expenses.ID},
				})).Return(&fxt.RegisterFXConversionResult{JournalEntryID: 42}, nil).Once()
			},
		},
		"from dollars to euros, without a note": {
			input: tt.RecordTransferInput{
				IdempotencyKey: "9c1e", FromLedgerAccountID: cashUSD.ID, ToLedgerAccountID: cash.ID,
				Amount: decimal.RequireFromString("324.15"), ToAmount: decimal.RequireFromString("300.50"), OccurredAt: occurredAt,
			},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(ledgerfxwf.RegisterFXConversion, mock.Anything, conversion(fxt.RegisterFXConversionInput{
					IdempotencyKey:                     "transfer:9c1e",
					OccurredAt:                         occurredAt,
					SourceLedgerAccountID:              cashUSD.ID,
					SourceClearingLedgerAccountID:      fxClearingUSD.ID,
					SourceAmount:                       decimal.RequireFromString("324.15"),
					DestinationLedgerAccountID:         cash.ID,
					DestinationClearingLedgerAccountID: fxClearing.ID,
					DestinationAmount:                  decimal.RequireFromString("300.50"),
				})).Return(&fxt.RegisterFXConversionResult{JournalEntryID: 42}, nil).Once()
			},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			mockLedgerAccounts(env)
			mockCategories(env)
			mockFXClearing(env)
			tc.mock(env)

			env.ExecuteWorkflow(RecordTransfer, tc.input)

			require.NoError(s.T(), env.GetWorkflowError())
			var got *tt.RecordResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), &tt.RecordResult{JournalEntryID: 42}, got)
			env.AssertExpectations(s.T())
		})
	}
}

// TestRecordTransfer_Errors checks that an invalid request, accounts that are
// not balances, amounts or charges that do not fit the balances' currencies,
// or a charge that is not an expense fail
// before anything is posted, since neither PostJournalEntry nor
// RegisterFXConversion is mocked unless a case reaches it, and that a failing
// child workflow's error is returned.
func (s *WorkflowUnitTestSuite) TestRecordTransfer_Errors() {
	input := func(from, to *model.LedgerAccount, toAmount string) tt.RecordTransferInput {
		return tt.RecordTransferInput{
			IdempotencyKey: "9c1e", FromLedgerAccountID: from.ID, ToLedgerAccountID: to.ID,
			Amount: decimal.NewFromInt(300), ToAmount: decimal.RequireFromString(toAmount),
			OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		input     tt.RecordTransferInput
		mock      func(env *testsuite.TestWorkflowEnvironment)
		wantError string
	}{
		"an invalid request": {
			input:     input(cash, cash, "0"),
			wantError: "invalid parameter: a transfer moves money between two different balances",
		},
		"a source that is not a balance": {
			input:     input(income, cash, "0"),
			wantError: `invalid parameter: "Income" is not a balance`,
		},
		"a destination of no financial account": {
			input:     input(cash, fxClearing, "0"),
			wantError: `invalid parameter: "FX clearing" is not a balance`,
		},
		"a different amount arriving in the same currency": {
			input:     input(cash, savingsVault, "299"),
			wantError: `invalid parameter: "Cash" and "Savings vault" are in the same currency`,
		},
		"no amount arriving in another currency": {
			input:     input(cash, cashUSD, "0"),
			wantError: `invalid parameter: "Cash" and "Cash (USD)" are in different currencies; give the amount that arrives as toAmount`,
		},
		"what arrives not less the charges": {
			input: func() tt.RecordTransferInput {
				i := input(cash, principal, "300")
				i.Charges = []tt.Split{{CategoryID: interest.ID, Amount: decimal.NewFromInt(100)}}
				return i
			}(),
			wantError: `invalid parameter: "Cash" and "Principal" are in the same currency, so the amount that arrives is the amount that leaves less the charges, 200`,
		},
		"a charge under an income category": {
			input: func() tt.RecordTransferInput {
				i := input(cash, principal, "0")
				i.Charges = []tt.Split{{CategoryID: salary.ID, Amount: decimal.NewFromInt(100)}}
				return i
			}(),
			wantError: `invalid parameter: "Salary" is an income category, not an expense one`,
		},
		"two charges between currencies": {
			input: func() tt.RecordTransferInput {
				i := input(cash, cashUSD, "324.15")
				i.Charges = []tt.Split{{CategoryID: interest.ID, Amount: decimal.NewFromInt(1)}, {CategoryID: groceries.ID, Amount: decimal.NewFromInt(1)}}
				return i
			}(),
			wantError: `invalid parameter: "Cash" and "Cash (USD)" are in different currencies, so the transfer takes at most one charge, its exchange fee`,
		},
		"an unknown account": {
			input: tt.RecordTransferInput{
				IdempotencyKey: "9c1e", FromLedgerAccountID: cash.ID, ToLedgerAccountID: 999,
				Amount: decimal.NewFromInt(300), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
			},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, act.RetrieveLedgerAccountParams{ID: 999}).
					Return(nil, errors.New("resource LedgerAccount not found with ID: 999"))
			},
			wantError: "retrieve ledger account 999",
		},
		"a failed posting": {
			input: input(cash, savingsVault, "0"),
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, mock.Anything).
					Return(nil, errors.New("resource LedgerAccount with ID: 102 is closed"))
			},
			wantError: "resource LedgerAccount with ID: 102 is closed",
		},
		"a failed clearing account lookup": {
			input: input(cash, cashUSD, "324.15"),
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, mock.Anything).
					Return(nil, errors.New("resource Currency not found with ID: 1"))
			},
			wantError: "find the FX clearing account of currency 1",
		},
		"a failed conversion": {
			input: input(cash, cashUSD, "324.15"),
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				mockFXClearing(env)
				env.OnWorkflow(ledgerfxwf.RegisterFXConversion, mock.Anything, mock.Anything).
					Return(nil, errors.New("resource LedgerAccount with ID: 103 is closed"))
			},
			wantError: "resource LedgerAccount with ID: 103 is closed",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			if tc.mock != nil {
				tc.mock(env)
			}
			mockLedgerAccounts(env)
			mockCategories(env)

			env.ExecuteWorkflow(RecordTransfer, tc.input)

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
