package mcptypes

import (
	"context"

	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

// WorkflowEmptyResult adapts a Temporal workflow function with no result
// into an MCP tool handler that runs it and waits for it to finish.
func WorkflowEmptyResult[T any, W any](c client.Client, wf W, opts ...workflowrun.Option) func(
	ctx context.Context,
	req *mcp.CallToolRequest, args T,
) (*mcp.CallToolResult, any, error) {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest, args T,
	) (*mcp.CallToolResult, any, error) {
		return nil, nil, workflowrun.RunWithoutResult(ctx, c, wf, args, opts...)
	}
}

// WorkflowWithResult adapts a Temporal workflow function into an MCP tool
// handler that runs it and returns its result.
//
// It is only a fit for a workflow whose input and output types have no bun
// relation fields: jsonschema-go's schema inference panics on a cyclic Go
// type rather than emitting a recursive schema, and most pkg/model types
// have one (FinancialAccount.LedgerAccounts <-> LedgerAccount.FinancialAccount,
// Asset.Transactions -> PositionMovement.Asset, ...). A tool whose domain
// result carries one of those needs a flat projection type and a handler of
// its own instead - see mcp/tools/accounts for an example.
func WorkflowWithResult[W any, I any, O any](c client.Client, workflow W, opts ...workflowrun.Option) func(
	ctx context.Context,
	req *mcp.CallToolRequest, args I,
) (*mcp.CallToolResult, *O, error) {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest, args I,
	) (*mcp.CallToolResult, *O, error) {
		result, err := workflowrun.Run[O](ctx, c, workflow, args, opts...)
		return nil, result, err
	}
}
