package accounts

import (
	"errors"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/workflow"
)

// ensureCategoryAccount runs EnsureCategoryAccount as a workflow would.
func ensureCategoryAccount(ctx workflow.Context, kind model.LedgerAccountKind, currencyID int64) (int64, error) {
	return EnsureCategoryAccount(ctx, kind, currencyID)
}

func (s *WorkflowUnitTestSuite) TestEnsureCategoryAccount() {
	tests := []struct {
		name      string
		kind      model.LedgerAccountKind
		wantName  string
		createErr string
		wantError string
	}{
		{name: "an expense", kind: model.LedgerAccountExpense, wantName: "Expenses"},
		{name: "an income", kind: model.LedgerAccountIncome, wantName: "Income"},
		{
			name: "an unknown currency", kind: model.LedgerAccountExpense, wantName: "Expenses",
			createErr: "resource Currency not found with ID: 2", wantError: "find the Expenses account of currency 2: ",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			env.RegisterWorkflow(ensureCategoryAccount)
			call := env.OnWorkflow(CreateLedgerAccount, mock.Anything, lt.CreateLedgerAccountInput{CurrencyID: 2, Name: tc.wantName, Kind: tc.kind})
			if tc.createErr != "" {
				call.Return(nil, errors.New(tc.createErr)).Once()
			} else {
				call.Return(&lt.CreateLedgerAccountResult{LedgerAccountID: 200}, nil).Once()
			}

			env.ExecuteWorkflow(ensureCategoryAccount, tc.kind, int64(2))

			if tc.wantError != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var got int64
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), int64(200), got)
			env.AssertExpectations(s.T())
		})
	}
}
