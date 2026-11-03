package assets

import (
	"context"

	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	assetswf "github.com/logicorg/ivfs/finance/workflows/assets"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

func registerListAssets(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "list_assets",
		Description: "Search tradable assets by a substring of their name and/or ticker, " +
			"to resolve a spoken reference like \"apple\" or \"AAPL\" to an asset id before calling a write command.",
	}, listAssetsHandler(c))
}

// listAssetsHandler does not use mcptypes.WorkflowWithResult: ListAssetsResult
// carries model.Asset values, whose relation fields form a cycle unusable as
// an MCP output schema (see mcptypes.WorkflowWithResult's doc comment), so
// this maps the result to AssetSummary by hand instead.
func listAssetsHandler(c client.Client) func(
	ctx context.Context, req *mcp.CallToolRequest, args at.ListAssetsParams,
) (*mcp.CallToolResult, *ListAssetsToolResult, error) {
	return func(
		ctx context.Context, req *mcp.CallToolRequest, args at.ListAssetsParams,
	) (*mcp.CallToolResult, *ListAssetsToolResult, error) {
		result, err := workflowrun.Run[at.ListAssetsResult](ctx, c, assetswf.ListAssets, args)
		if err != nil {
			return nil, nil, err
		}

		out := &ListAssetsToolResult{Assets: make([]AssetSummary, len(result.Assets))}
		for i, a := range result.Assets {
			out.Assets[i] = AssetSummary{
				ID: a.ID, Name: a.Name, Kind: string(a.Kind),
				Ticker: a.Ticker, ISIN: a.ISIN, Exchange: a.Exchange,
			}
		}
		return nil, out, nil
	}
}
