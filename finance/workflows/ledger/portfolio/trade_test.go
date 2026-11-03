package portfolio

import (
	"github.com/google/uuid"
	activity "github.com/logicorg/ivfs/finance/activities/ledger/portfolio"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestRegisterInvestmentTrade() {
	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{
			name: "success",
			result: []any{
				&lt.RegisterInvestmentTradeResult{
					JournalEntryID:     10,
					PositionMovementID: 20,
				},
				nil,
			},
		},
		{
			name: "activity failure",
			result: []any{
				nil,
				assert.AnError,
			},
			wantErr: true,
		},
	}

	input := lt.RegisterInvestmentTradeInput{
		Entry:              lt.PostJournalEntryInput{IdempotencyKey: "buy-aapl-1"},
		FinancialAccountID: 1,
		AssetID:            2,
		UnitsDelta:         decimal.NewFromInt(1),
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			pa := activity.Activities{}

			env.OnActivity(pa.RegisterInvestmentTrade, mock.Anything, input).Return(tc.result...)

			env.ExecuteWorkflow(RegisterInvestmentTrade, input)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.RegisterInvestmentTradeResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result[0], result)
		})
	}
}

// TestRegisterInvestmentTradeDefaultsIdempotencyKey checks that a caller
// leaving Entry.IdempotencyKey empty gets a generated one filled in before
// the activity is called, and that two independent executions produce two
// different generated keys.
func (s *WorkflowUnitTestSuite) TestRegisterInvestmentTradeDefaultsIdempotencyKey() {
	input := lt.RegisterInvestmentTradeInput{FinancialAccountID: 1, AssetID: 2, UnitsDelta: decimal.NewFromInt(1)}

	var generated [2]string
	for i := range generated {
		env := s.NewTestWorkflowEnvironment()
		pa := activity.Activities{}

		env.OnActivity(pa.RegisterInvestmentTrade, mock.Anything, mock.MatchedBy(func(got lt.RegisterInvestmentTradeInput) bool {
			generated[i] = got.Entry.IdempotencyKey
			return true
		})).Return(&lt.RegisterInvestmentTradeResult{JournalEntryID: 1, PositionMovementID: 1}, nil)

		env.ExecuteWorkflow(RegisterInvestmentTrade, input)
		require.NoError(s.T(), env.GetWorkflowError())
	}

	for _, key := range generated {
		_, err := uuid.Parse(key)
		require.NoError(s.T(), err)
	}
	require.NotEqual(s.T(), generated[0], generated[1])
}
