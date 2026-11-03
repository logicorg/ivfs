package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestPreviewCategoryDeletion() {
	preview := &lt.AccountDeletionPreview{DeletedEntries: 3, KeptEntries: 1}

	tests := []struct {
		name  string
		input ct.PreviewCategoryDeletionInput
		// kind is the retrieved account's kind; empty means it does not exist.
		kind     model.LedgerAccountKind
		retrieve bool
		previews bool
		wantErr  string
	}{
		{
			name:     "an expense account",
			input:    ct.PreviewCategoryDeletionInput{LedgerAccountID: 7},
			kind:     model.LedgerAccountExpense,
			retrieve: true,
			previews: true,
		},
		{
			name:     "an account that does not exist",
			input:    ct.PreviewCategoryDeletionInput{LedgerAccountID: 7},
			retrieve: true,
			wantErr:  "resource LedgerAccount not found with ID: 7",
		},
		{
			name:     "an asset account",
			input:    ct.PreviewCategoryDeletionInput{LedgerAccountID: 7},
			kind:     model.LedgerAccountAsset,
			retrieve: true,
			wantErr:  "ledger account 7 has kind asset, not income or expense; delete it as a ledger account",
		},
		{
			name:    "an id that is not positive",
			input:   ct.PreviewCategoryDeletionInput{},
			wantErr: "validation error",
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			t := s.T()
			env := s.NewTestWorkflowEnvironment()
			env.RegisterWorkflow(ledgeraccountswf.RetrieveLedgerAccount)
			env.RegisterWorkflow(ledgeraccountswf.PreviewAccountDeletion)

			if tc.retrieve {
				retrieved := env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: tc.input.LedgerAccountID})
				if tc.kind == "" {
					retrieved.Return(nil, internalerrors.ErrNotFound{ResourceType: "LedgerAccount", ID: tc.input.LedgerAccountID}).Once()
				} else {
					retrieved.Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: &model.LedgerAccount{ID: tc.input.LedgerAccountID, Kind: tc.kind}}, nil).Once()
				}
			}
			if tc.previews {
				env.OnWorkflow(ledgeraccountswf.PreviewAccountDeletion, mock.Anything, lt.PreviewAccountDeletionInput{LedgerAccountID: &tc.input.LedgerAccountID}).
					Return(preview, nil).Once()
			}

			env.ExecuteWorkflow(PreviewCategoryDeletion, tc.input)

			env.AssertExpectations(t)
			if tc.wantErr != "" {
				require.ErrorContains(t, env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(t, env.GetWorkflowError())
			var got lt.AccountDeletionPreview
			require.NoError(t, env.GetWorkflowResult(&got))
			require.Equal(t, *preview, got)
		})
	}
}
