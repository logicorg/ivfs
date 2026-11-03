package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestRenameAndMoveCategories() {
	renameInput := ct.RenameCategoryInput{LedgerAccountID: 10, Name: "Food"}
	env := s.NewTestWorkflowEnvironment()
	env.OnWorkflow(ledgeraccountswf.RenameLedgerAccount, mock.Anything, lt.RenameLedgerAccountInput{LedgerAccountID: 10, Name: "Food"}).Return(&lt.LedgerAccountMutationResult{LedgerAccountID: 10}, nil)
	env.ExecuteWorkflow(RenameCategory, renameInput)
	require.NoError(s.T(), env.GetWorkflowError())

	moveInput := ct.MoveCategoryInput{LedgerAccountID: 10, ParentID: ptr.Int64(11)}
	env = s.NewTestWorkflowEnvironment()
	env.OnWorkflow(ledgeraccountswf.MoveLedgerAccount, mock.Anything, lt.MoveLedgerAccountInput{LedgerAccountID: 10, ParentID: ptr.Int64(11)}).Return(&lt.LedgerAccountMutationResult{LedgerAccountID: 10}, nil)
	env.ExecuteWorkflow(MoveCategory, moveInput)
	require.NoError(s.T(), env.GetWorkflowError())
}
