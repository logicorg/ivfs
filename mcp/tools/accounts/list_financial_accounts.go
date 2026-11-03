package accounts

import (
	"context"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

func registerListFinancialAccounts(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "list_financial_accounts",
		Description: "Search financial accounts by a substring of their own name and/or their institution's name, " +
			"to resolve a spoken reference like \"revolut eur\" to a financial account id before calling a write command.",
	}, listFinancialAccountsHandler(c))
}

// listFinancialAccountsHandler does not use mcptypes.WorkflowWithResult:
// ListFinancialAccountsResult carries model.FinancialAccount values, whose
// relation fields form a cycle unusable as an MCP output schema (see
// mcptypes.WorkflowWithResult's doc comment), so this maps the result to
// FinancialAccountSummary by hand instead.
func listFinancialAccountsHandler(c client.Client) func(
	ctx context.Context, req *mcp.CallToolRequest, args lt.ListFinancialAccountsParams,
) (*mcp.CallToolResult, *ListFinancialAccountsToolResult, error) {
	return func(
		ctx context.Context, req *mcp.CallToolRequest, args lt.ListFinancialAccountsParams,
	) (*mcp.CallToolResult, *ListFinancialAccountsToolResult, error) {
		result, err := workflowrun.Run[lt.ListFinancialAccountsResult](ctx, c, ledgeraccountswf.ListFinancialAccounts, args)
		if err != nil {
			return nil, nil, err
		}

		out := &ListFinancialAccountsToolResult{FinancialAccounts: make([]FinancialAccountSummary, len(result.FinancialAccounts))}
		for i, fa := range result.FinancialAccounts {
			s := FinancialAccountSummary{ID: fa.ID, Name: fa.Name, InstitutionID: fa.InstitutionID, Closed: fa.ClosedAt != nil}
			if fa.Institution != nil {
				s.InstitutionName = fa.Institution.Name
			}
			out.FinancialAccounts[i] = s
		}
		return nil, out, nil
	}
}
