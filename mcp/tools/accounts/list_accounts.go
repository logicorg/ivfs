package accounts

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// listAccountsTimeout bounds how long a list_accounts call waits. The
// workflow only reads, so anything longer means it is stuck, such as on a
// panic Temporal keeps retrying.
const listAccountsTimeout = 30 * time.Second

func registerListAccounts(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "list_accounts",
		Description: "List ledger accounts with their kind, currency code, and the names of the financial account and " +
			"institution they belong to, such as the EUR Cash account of Revolut Personal, or a category such as Groceries. " +
			"Use it to resolve a spoken reference like \"revolut eur\" or \"groceries\" to the ledger account id that " +
			"post_journal_entry, register_fx_conversion, and get_ledger_account_balance need. Closed accounts are left out " +
			"unless asked for.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.ListAccountsParams) (*lt.ListAccountsResult, error),
		lt.ListAccountsParams, lt.ListAccountsResult,
	](c, ledgeraccountswf.ListAccounts, func(o *client.StartWorkflowOptions) {
		o.WorkflowExecutionTimeout = listAccountsTimeout
	}))
}
