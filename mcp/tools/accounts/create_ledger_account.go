package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerCreateLedgerAccount(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "create_ledger_account",
		Description: "Create a ledger account: one balance in one currency. Under a financial account it is, for example, the cash " +
			"in another currency, a savings vault, or a card balance (a liability); without one it is an equity account such as " +
			"opening balances, or a per-currency FX clearing account (an asset). For income and expense categories use " +
			"create_category instead. Returns the existing account instead when one already has the same name, compared " +
			"case-insensitively, under the same financial account and currency, or, without a financial account, under the same " +
			"parent with the same kind and currency. Get ids with list_financial_accounts and list_currencies.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.CreateLedgerAccountInput) (*lt.CreateLedgerAccountResult, error),
		lt.CreateLedgerAccountInput, lt.CreateLedgerAccountResult,
	](c, ledgeraccountswf.CreateLedgerAccount))
}
