package fx

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	fxwf "github.com/logicorg/ivfs/finance/workflows/ledger/fx"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerRegisterFXConversion(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "register_fx_conversion",
		Description: "Convert money between two currencies as one balanced journal entry, bridged by a clearing account per currency. " +
			"Both amounts are what actually settled - the rate is derived and stored, not supplied. Look ledger account ids up first " +
			"with list_accounts rather than guessing them.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.RegisterFXConversionInput) (*lt.RegisterFXConversionResult, error),
		RegisterFXConversionToolInput, lt.RegisterFXConversionResult,
	](c, fxwf.RegisterFXConversion))
}
