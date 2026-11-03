package portfolio

import (
	"time"

	"github.com/google/uuid"
	activity "github.com/logicorg/ivfs/finance/activities/ledger/portfolio"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestRegisterVest() {
	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{
			name:   "success",
			result: []any{&lt.RegisterVestResult{JournalEntryID: 10, PositionMovementID: 20}, nil},
		},
		{
			name:    "activity failure",
			result:  []any{nil, assert.AnError},
			wantErr: true,
		},
	}

	input := lt.RegisterVestInput{
		FinancialAccountID:  1,
		AssetID:             2,
		Units:               decimal.NewFromInt(60),
		CostBasisAmount:     decimal.NewFromInt(600),
		CostBasisCurrencyID: 3,
		Entry: lt.PostJournalEntryInput{
			IdempotencyKey: "acme-rsu-1",
			OccurredAt:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
			Postings: []lt.PostingInput{
				{LedgerAccountID: 10, Amount: decimal.NewFromInt(600), Side: model.PostingCredit},
				{LedgerAccountID: 20, Amount: decimal.NewFromInt(600), Side: model.PostingDebit},
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			pa := activity.Activities{}

			env.OnActivity(pa.RegisterVest, mock.Anything, input).Return(tc.result...)

			env.ExecuteWorkflow(RegisterVest, input)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.RegisterVestResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result[0], result)
		})
	}
}

// TestRegisterVestDefaultsIdempotencyKey checks that a caller leaving
// Entry.IdempotencyKey empty gets a generated one filled in before the
// activity is called, and that two independent executions produce two
// different generated keys.
func (s *WorkflowUnitTestSuite) TestRegisterVestDefaultsIdempotencyKey() {
	input := lt.RegisterVestInput{
		FinancialAccountID:  1,
		AssetID:             2,
		Units:               decimal.NewFromInt(60),
		CostBasisAmount:     decimal.NewFromInt(600),
		CostBasisCurrencyID: 3,
	}

	var generated [2]string
	for i := range generated {
		env := s.NewTestWorkflowEnvironment()
		pa := activity.Activities{}

		env.OnActivity(pa.RegisterVest, mock.Anything, mock.MatchedBy(func(got lt.RegisterVestInput) bool {
			generated[i] = got.Entry.IdempotencyKey
			return true
		})).Return(&lt.RegisterVestResult{JournalEntryID: 1, PositionMovementID: 1}, nil)

		env.ExecuteWorkflow(RegisterVest, input)
		require.NoError(s.T(), env.GetWorkflowError())
	}

	for _, key := range generated {
		_, err := uuid.Parse(key)
		require.NoError(s.T(), err)
	}
	require.NotEqual(s.T(), generated[0], generated[1])
}
