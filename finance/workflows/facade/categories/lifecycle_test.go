package categories

import (
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestCloseCategory() {
	closeInput := ct.CloseCategoryInput{LedgerAccountID: 10, ClosedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	env := s.NewTestWorkflowEnvironment()
	env.OnWorkflow(ledgeraccountswf.CloseLedgerAccount, mock.Anything, lt.CloseLedgerAccountInput{LedgerAccountID: 10, ClosedAt: closeInput.ClosedAt}).Return(nil)
	env.ExecuteWorkflow(CloseCategory, closeInput)
	require.NoError(s.T(), env.GetWorkflowError())
}
