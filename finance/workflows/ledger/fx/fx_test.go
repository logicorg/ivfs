package fx

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	ledgerfx "github.com/logicorg/ivfs/finance/activities/ledger/fx"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	fxt "github.com/logicorg/ivfs/finance/common/types/fxtypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type WorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestWorkflowUnitTestSuite(t *testing.T) { suite.Run(t, new(WorkflowUnitTestSuite)) }

func (s *WorkflowUnitTestSuite) TestRegisterFXConversion() {
	input := fxt.RegisterFXConversionInput{
		IdempotencyKey:                     "fx-usd-eur-1",
		OccurredAt:                         time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
		SourceLedgerAccountID:              1,
		SourceClearingLedgerAccountID:      2,
		SourceAmount:                       decimal.NewFromInt(100),
		DestinationLedgerAccountID:         3,
		DestinationClearingLedgerAccountID: 4,
		DestinationAmount:                  decimal.NewFromInt(90),
	}

	tests := map[string]struct {
		result  []any
		wantErr bool
	}{
		"success":          {result: []any{&fxt.RegisterFXConversionResult{JournalEntryID: 7}, nil}},
		"activity failure": {result: []any{nil, assert.AnError}, wantErr: true},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := ledgerfx.Activities{}
			env.OnActivity(aa.RegisterFXConversion, mock.Anything, input).Return(tc.result...)

			env.ExecuteWorkflow(RegisterFXConversion, input)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *fxt.RegisterFXConversionResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result[0], result)
		})
	}
}

// TestRegisterFXConversion_Fee checks that a fee is checked to be under an
// expense category, closed or not, and reaches the activity on the "Expenses"
// account of the source currency, whatever account the caller sent; and that
// a fee that cannot be booked fails before the activity, which is mocked only
// where a case reaches it.
func (s *WorkflowUnitTestSuite) TestRegisterFXConversion_Fee() {
	const (
		revolutUSD  int64 = 1
		unknownID   int64 = 999
		usd         int64 = 2
		usdExpenses int64 = 200
	)
	bankFees := &model.Category{ID: 20, Name: "Bank fees", Kind: model.LedgerAccountExpense}
	oldFees := &model.Category{ID: 21, Name: "Old fees", Kind: model.LedgerAccountExpense, ClosedAt: ptr.NewPtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))}
	interest := &model.Category{ID: 30, Name: "Interest", Kind: model.LedgerAccountIncome}

	conversion := func(source int64, fee fxt.FXFee) fxt.RegisterFXConversionInput {
		return fxt.RegisterFXConversionInput{
			IdempotencyKey:                     "fx-usd-eur-1",
			OccurredAt:                         time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
			SourceLedgerAccountID:              source,
			SourceClearingLedgerAccountID:      2,
			SourceAmount:                       decimal.NewFromInt(100),
			DestinationLedgerAccountID:         3,
			DestinationClearingLedgerAccountID: 4,
			DestinationAmount:                  decimal.NewFromInt(90),
			Fee:                                &fee,
		}
	}
	one := decimal.NewFromInt(1)

	tests := []struct {
		name  string
		input fxt.RegisterFXConversionInput
		// want is the fee the activity receives, nil when it is not reached.
		want      *fxt.FXFee
		wantError string
	}{
		{
			name:  "a fee under an expense category",
			input: conversion(revolutUSD, fxt.FXFee{CategoryID: bankFees.ID, Amount: one}),
			want:  &fxt.FXFee{CategoryID: bankFees.ID, Amount: one, LedgerAccountID: usdExpenses},
		},
		{
			name:  "a fee under a closed category, with an account the caller sent",
			input: conversion(revolutUSD, fxt.FXFee{CategoryID: oldFees.ID, Amount: one, LedgerAccountID: 77}),
			want:  &fxt.FXFee{CategoryID: oldFees.ID, Amount: one, LedgerAccountID: usdExpenses},
		},
		{
			name:      "a fee without a category",
			input:     conversion(revolutUSD, fxt.FXFee{Amount: one}),
			wantError: "missing required parameters: Fee.CategoryID",
		},
		{
			name:      "a fee under an income category",
			input:     conversion(revolutUSD, fxt.FXFee{CategoryID: interest.ID, Amount: one}),
			wantError: `invalid parameter: "Interest" is an income category, not an expense one`,
		},
		{
			name:      "a fee under an unknown category",
			input:     conversion(revolutUSD, fxt.FXFee{CategoryID: unknownID, Amount: one}),
			wantError: "retrieve category 999: ",
		},
		{
			name:      "an unknown source account",
			input:     conversion(unknownID, fxt.FXFee{CategoryID: bankFees.ID, Amount: one}),
			wantError: "retrieve ledger account 999: ",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			for _, category := range []*model.Category{bankFees, oldFees, interest} {
				env.OnWorkflow(categorieswf.RetrieveCategory, mock.Anything, ct.RetrieveCategoryParams{ID: category.ID}).
					Return(&ct.RetrieveCategoryResult{Category: category}, nil).Maybe()
			}
			env.OnWorkflow(categorieswf.RetrieveCategory, mock.Anything, ct.RetrieveCategoryParams{ID: unknownID}).
				Return(nil, errors.New("resource Category not found with ID: 999")).Maybe()
			env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: revolutUSD}).
				Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: &model.LedgerAccount{ID: revolutUSD, CurrencyID: usd, Name: "Revolut USD", Kind: model.LedgerAccountAsset}}, nil).Maybe()
			env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: unknownID}).
				Return(nil, errors.New("resource LedgerAccount not found with ID: 999")).Maybe()
			env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, lt.CreateLedgerAccountInput{
				CurrencyID: usd, Name: "Expenses", Kind: model.LedgerAccountExpense,
			}).Return(&lt.CreateLedgerAccountResult{LedgerAccountID: usdExpenses}, nil).Maybe()
			if tc.want != nil {
				aa := ledgerfx.Activities{}
				env.OnActivity(aa.RegisterFXConversion, mock.Anything, mock.MatchedBy(func(got fxt.RegisterFXConversionInput) bool {
					return got.Fee != nil && got.Fee.CategoryID == tc.want.CategoryID &&
						got.Fee.LedgerAccountID == tc.want.LedgerAccountID && got.Fee.Amount.Equal(tc.want.Amount)
				})).Return(&fxt.RegisterFXConversionResult{JournalEntryID: 7}, nil).Once()
			}

			env.ExecuteWorkflow(RegisterFXConversion, tc.input)

			if tc.wantError != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *fxt.RegisterFXConversionResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), int64(7), result.JournalEntryID)
			env.AssertExpectations(s.T())
		})
	}
}

// TestRegisterFXConversionDefaultsIdempotencyKey checks that a caller
// leaving IdempotencyKey empty gets a generated one filled in before the
// activity is called, and that two independent executions produce two
// different generated keys.
func (s *WorkflowUnitTestSuite) TestRegisterFXConversionDefaultsIdempotencyKey() {
	input := fxt.RegisterFXConversionInput{
		OccurredAt:                         time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
		SourceLedgerAccountID:              1,
		SourceClearingLedgerAccountID:      2,
		SourceAmount:                       decimal.NewFromInt(100),
		DestinationLedgerAccountID:         3,
		DestinationClearingLedgerAccountID: 4,
		DestinationAmount:                  decimal.NewFromInt(90),
	}

	var generated [2]string
	for i := range generated {
		env := s.NewTestWorkflowEnvironment()
		aa := ledgerfx.Activities{}

		env.OnActivity(aa.RegisterFXConversion, mock.Anything, mock.MatchedBy(func(got fxt.RegisterFXConversionInput) bool {
			generated[i] = got.IdempotencyKey
			return true
		})).Return(&fxt.RegisterFXConversionResult{JournalEntryID: 1}, nil)

		env.ExecuteWorkflow(RegisterFXConversion, input)
		require.NoError(s.T(), env.GetWorkflowError())
	}

	for _, key := range generated {
		_, err := uuid.Parse(key)
		require.NoError(s.T(), err)
	}
	require.NotEqual(s.T(), generated[0], generated[1])
}
