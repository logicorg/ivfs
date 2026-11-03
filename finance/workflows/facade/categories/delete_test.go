package categories

import (
	"github.com/google/uuid"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

func (s *WorkflowUnitTestSuite) TestDeleteCategory() {
	user := uuid.MustParse("0192f3a0-0000-7000-8000-000000000001")
	tests := []struct {
		name  string
		input ct.DeleteCategoryInput
		// kind is the retrieved account's kind; empty means it does not exist.
		kind     model.LedgerAccountKind
		retrieve bool
		// deletes is whether DeleteLedgerAccount runs, and deleteErr what it
		// returns.
		deletes   bool
		deleteErr error
		wantErr   string
	}{
		{
			name:    "an id that is not positive",
			input:   ct.DeleteCategoryInput{},
			wantErr: "validation error",
		},
		{
			name:     "an expense account",
			input:    ct.DeleteCategoryInput{LedgerAccountID: 7},
			kind:     model.LedgerAccountExpense,
			retrieve: true,
			deletes:  true,
		},
		{
			name:     "an income account merged into another",
			input:    ct.DeleteCategoryInput{LedgerAccountID: 7, ReplacementID: ptr.Int64(8)},
			kind:     model.LedgerAccountIncome,
			retrieve: true,
			deletes:  true,
		},
		{
			name:     "an account that does not exist",
			input:    ct.DeleteCategoryInput{LedgerAccountID: 7},
			retrieve: true,
		},
		{
			name:     "an asset account",
			input:    ct.DeleteCategoryInput{LedgerAccountID: 7},
			kind:     model.LedgerAccountAsset,
			retrieve: true,
			wantErr:  "ledger account 7 has kind asset, not income or expense; delete it as a ledger account",
		},
		{
			name:      "a deletion that fails",
			input:     ct.DeleteCategoryInput{LedgerAccountID: 7},
			kind:      model.LedgerAccountExpense,
			retrieve:  true,
			deletes:   true,
			deleteErr: internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account 7 has child accounts; delete or move them first"},
			wantErr:   "resource LedgerAccount conflict: ledger account 7 has child accounts; delete or move them first",
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			t := s.T()
			env := s.NewTestWorkflowEnvironment()
			temporaltesting.RunAs(t, env, user)
			env.RegisterWorkflow(ledgeraccountswf.RetrieveLedgerAccount)
			env.RegisterWorkflow(ledgeraccountswf.DeleteLedgerAccount)

			if tc.retrieve {
				retrieved := env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: tc.input.LedgerAccountID})
				if tc.kind == "" {
					retrieved.Return(nil, internalerrors.ErrNotFound{ResourceType: "LedgerAccount", ID: tc.input.LedgerAccountID}).Once()
				} else {
					retrieved.Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: &model.LedgerAccount{ID: tc.input.LedgerAccountID, Kind: tc.kind}}, nil).Once()
				}
			}
			var deletionIDs []string
			if tc.deletes {
				env.OnWorkflow(ledgeraccountswf.DeleteLedgerAccount, mock.Anything, lt.DeleteLedgerAccountInput{
					LedgerAccountID: tc.input.LedgerAccountID, ReplacementID: tc.input.ReplacementID,
				}).Return(&lt.DeleteAccountResult{}, tc.deleteErr).Once()
				env.SetOnChildWorkflowStartedListener(func(info *workflow.Info, _ workflow.Context, _ converter.EncodedValues) {
					if info.WorkflowType.Name == "DeleteLedgerAccount" {
						deletionIDs = append(deletionIDs, info.WorkflowExecution.ID)
					}
				})
			}

			env.ExecuteWorkflow(DeleteCategory, tc.input)

			env.AssertExpectations(t)
			if tc.deletes {
				require.Equal(t, []string{tenancy.WorkflowIDPrefix(user) + workflows.DeleteLedgerAccountWorkflowID(tc.input.LedgerAccountID)}, deletionIDs)
			}
			if tc.wantErr != "" {
				require.ErrorContains(t, env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(t, env.GetWorkflowError())
			var result lt.DeleteAccountResult
			require.NoError(t, env.GetWorkflowResult(&result))
		})
	}
}
