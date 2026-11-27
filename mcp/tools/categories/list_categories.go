package categories

import (
	"context"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

func registerListCategories(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "list_categories",
		Description: "Search income/expense categories by a substring of their name, " +
			"to resolve a spoken reference like \"groceries\" to a category id before calling a write command. " +
			"Closed categories are left out unless asked for.",
	}, listCategoriesHandler(c))
}

// listCategoriesHandler does not use mcptypes.WorkflowWithResult:
// ListCategoriesResult carries model.Category values, whose relation fields
// form a cycle unusable as an MCP output schema (see
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

		out := &ListCategoriesToolResult{Categories: make([]CategorySummary, len(result.Categories))}
		for i, category := range result.Categories {
			out.Categories[i] = CategorySummary{
				ID: category.ID, Name: category.Name, ParentID: category.ParentID,
				Kind: string(category.Kind), Closed: category.ClosedAt != nil,
			}
		}
		return nil, out, nil
	}
}
