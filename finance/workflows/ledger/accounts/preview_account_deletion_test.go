package accounts

import (
	accountdeletion "github.com/logicorg/ivfs/finance/activities/ledger/accounts/deletion"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestPreviewAccountDeletion() {
	preview := &act.AccountDeletionPreview{DeletedEntries: 2, KeptEntries: 1, Conflicts: []string{"ledger account 7 has child accounts; delete them first"}}

	tests := []struct {
		name  string
		input act.PreviewAccountDeletionInput
		// previews is whether the activity runs, and previewErr what it returns.
		previews   bool
		previewErr error
		want       *act.AccountDeletionPreview
		wantErr    string
	}{
		{
			name:     "a ledger account",
			input:    act.PreviewAccountDeletionInput{LedgerAccountID: ptr.Int64(7)},
			previews: true,
			want:     preview,
		},
		{
			name:       "an institution that does not exist",
			input:      act.PreviewAccountDeletionInput{InstitutionID: ptr.Int64(7)},
			previews:   true,
			previewErr: internalerrors.ErrNotFound{ResourceType: "Institution", ID: 7},
			wantErr:    "resource Institution not found with ID: 7",
		},
		{
			name:    "no target",
			input:   act.PreviewAccountDeletionInput{},
			wantErr: "a deletion preview takes exactly one of a ledger account, a financial account, or an institution",
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			t := s.T()
			env := s.NewTestWorkflowEnvironment()
			da := accountdeletion.Activities{}
			if tc.previews {
				env.OnActivity(da.PreviewAccountDeletion, mock.Anything, tc.input).Return(preview, tc.previewErr).Once()
			}

			env.ExecuteWorkflow(PreviewAccountDeletion, tc.input)

			env.AssertExpectations(t)
			if tc.wantErr != "" {
				require.ErrorContains(t, env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(t, env.GetWorkflowError())
			var got act.AccountDeletionPreview
			require.NoError(t, env.GetWorkflowResult(&got))
			require.Equal(t, *tc.want, got)
		})
	}
}
