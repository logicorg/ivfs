package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerCreateFinancialAccount(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "create_financial_account",
		Description: "Create a financial account: one relationship with an institution, such as a personal, joint, or business " +
			"account, a card, or a brokerage account. It has no currency of its own; its balances are ledger accounts. With " +
			"cashCurrencyId, a \"Cash\" ledger account in that currency is created with it; add other currencies with " +
			"create_ledger_account. Returns the existing account instead when the institution already has one with that name, " +
			"compared case-insensitively. Get the institution id with create_institution and currency ids with list_currencies.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.CreateFinancialAccountInput) (*lt.CreateFinancialAccountResult, error),
		lt.CreateFinancialAccountInput, lt.CreateFinancialAccountResult,
	](c, ledgeraccountswf.CreateFinancialAccount))
}
