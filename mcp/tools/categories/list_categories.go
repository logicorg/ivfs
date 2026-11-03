package categories

import (
	"context"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

func registerListCategories(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "list_categories",
		Description: "Search income/expense categories by a substring of their name, " +
			"to resolve a spoken reference like \"groceries\" to a ledger account id before calling a write command.",
	}, listCategoriesHandler(c))
}

// listCategoriesHandler does not use mcptypes.WorkflowWithResult:
// ListCategoriesResult carries model.LedgerAccount values, whose relation
// fields form a cycle unusable as an MCP output schema (see
// mcptypes.WorkflowWithResult's doc comment), so this maps the result to
// CategorySummary by hand instead.
func listCategoriesHandler(c client.Client) func(
	ctx context.Context, req *mcp.CallToolRequest, args ct.ListCategoriesParams,
) (*mcp.CallToolResult, *ListCategoriesToolResult, error) {
	return func(
		ctx context.Context, req *mcp.CallToolRequest, args ct.ListCategoriesParams,
	) (*mcp.CallToolResult, *ListCategoriesToolResult, error) {
		result, err := workflowrun.Run[ct.ListCategoriesResult](ctx, c, categorieswf.ListCategories, args)
		if err != nil {
			return nil, nil, err
		}

		out := &ListCategoriesToolResult{Categories: make([]CategorySummary, len(result.LedgerAccounts))}
		for i, la := range result.LedgerAccounts {
			out.Categories[i] = CategorySummary{
				ID: la.ID, Name: la.Name, ParentID: la.ParentID,
				CurrencyID: la.CurrencyID, Kind: string(la.Kind), Closed: la.ClosedAt != nil,
			}
		}
		return nil, out, nil
	}
}
