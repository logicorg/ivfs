package reporting

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerGetFinancialAccountBalances(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "get_financial_account_balances",
		Description: "Get the balance of every ledger account under a financial account, such as its cash in each currency, " +
			"each with the ledger account's id, name, kind, and currency code. Answers \"how much is in my Revolut account?\", " +
			"and also finds the ledger account ids under a financial account that post_journal_entry needs. Only posted " +
			"entries count. A balance is in the account's natural direction, so a positive liability is money owed. Get the " +
			"financial account id with list_financial_accounts.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.FinancialAccountBalancesParams) (*lt.FinancialAccountBalancesResult, error),
		lt.FinancialAccountBalancesParams, lt.FinancialAccountBalancesResult,
	](c, reportingwf.FinancialAccountBalances))
}
