package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerDeleteLedgerAccount(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-ledger-account",
		Method:      http.MethodDelete,
		Path:        "/api/ledger-accounts/{id}",
		Summary:     "Delete a ledger account with the entries and movements recorded on it",
		Description: "An entry that also moves money in other financial accounts is kept, with this account's lines moved to " +
			"the untracked accounts. Deleting an account that does not exist changes nothing. A deletion still running " +
			"after a while is answered with 202 Accepted, and deleting again waits on it.",
		Tags:      tags,
		Errors:    []int{http.StatusConflict, http.StatusUnprocessableEntity},
		Responses: gateway.AccountDeletionResponses(api),
	}, func(ctx context.Context, in *DeleteAccountInput) (*gateway.AccountDeletionOutput, error) {
		return gateway.DeleteAccount(ctx, c, workflows.DeleteLedgerAccountWorkflowID(in.ID), ledgeraccountswf.DeleteLedgerAccount,
			lt.DeleteLedgerAccountInput{LedgerAccountID: in.ID, ReplacementID: gateway.Replacement(in.ReplacementID)},
			func(result *lt.DeleteAccountResult) []lt.NegativeHolding { return result.Warnings })
	})
}
