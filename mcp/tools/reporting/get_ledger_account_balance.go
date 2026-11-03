package reporting

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerGetLedgerAccountBalance(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "get_ledger_account_balance",
		Description: "Get the balance of one ledger account, with its name, kind, and currency code: a cash account, a card, " +
			"or a category such as \"Groceries\", where the balance is the total spent or earned. Only posted entries count. " +
			"A balance is in the account's natural direction: debits minus credits for assets and expenses, credits minus " +
			"debits for liabilities, equity, and income, so each is positive in normal use. Get the id with list_accounts.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.LedgerAccountBalanceParams) (*lt.LedgerAccountBalanceResult, error),
		lt.LedgerAccountBalanceParams, lt.LedgerAccountBalanceResult,
	](c, reportingwf.LedgerAccountBalance))
}
