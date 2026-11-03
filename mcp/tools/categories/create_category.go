package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerCreateCategory(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "create_category",
		Description: "Create an income or expense category, such as \"Groceries\" or \"Salary\", optionally under a parent category " +
			"of the same kind and currency (at most three levels deep). Returns the existing category instead when the parent " +
			"already has one with the same name, kind, and currency, compared case-insensitively. Look the currency up with " +
			"list_currencies and parents with list_categories.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, ct.CreateCategoryInput) (*ct.CategoryResult, error),
		ct.CreateCategoryInput, ct.CategoryResult,
	](c, categorieswf.CreateCategory))
}
