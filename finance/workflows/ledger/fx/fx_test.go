package fx

import (
	"testing"
	"time"

	"github.com/google/uuid"
	ledgerfx "github.com/logicorg/ivfs/finance/activities/ledger/fx"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
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
	input := lt.RegisterFXConversionInput{
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
		"success":          {result: []any{&lt.RegisterFXConversionResult{JournalEntryID: 7}, nil}},
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
			var result *lt.RegisterFXConversionResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result[0], result)
		})
	}
}

// TestRegisterFXConversionDefaultsIdempotencyKey checks that a caller
// leaving IdempotencyKey empty gets a generated one filled in before the
// activity is called, and that two independent executions produce two
// different generated keys.
func (s *WorkflowUnitTestSuite) TestRegisterFXConversionDefaultsIdempotencyKey() {
	input := lt.RegisterFXConversionInput{
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

		env.OnActivity(aa.RegisterFXConversion, mock.Anything, mock.MatchedBy(func(got lt.RegisterFXConversionInput) bool {
			generated[i] = got.IdempotencyKey
			return true
		})).Return(&lt.RegisterFXConversionResult{JournalEntryID: 1}, nil)

		env.ExecuteWorkflow(RegisterFXConversion, input)
		require.NoError(s.T(), env.GetWorkflowError())
	}

	for _, key := range generated {
		_, err := uuid.Parse(key)
		require.NoError(s.T(), err)
	}
	require.NotEqual(s.T(), generated[0], generated[1])
}
