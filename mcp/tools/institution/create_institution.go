package institution

import (
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerCreateInstitution(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "create_institution",
		Description: "Create an institution (a bank, broker, card issuer, or other provider), or return the existing one with the same " +
			"name, compared case-insensitively. Kind is free text such as \"bank\" or \"broker\". Safe to call again: to find an " +
			"institution's id, call this with its name.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, it.InsertInstitutionParams) (*it.InsertInstitutionResult, error),
		it.InsertInstitutionParams, it.InsertInstitutionResult,
	](c, institutionwf.InsertInstitution))
}
