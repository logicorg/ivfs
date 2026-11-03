package categories

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerDelete(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-category",
		Method:      http.MethodDelete,
		Path:        "/api/categories/{id}",
		Summary:     "Delete a category with the entries recorded on it",
		Description: "An entry that also moves money in a financial account is kept, with this category's lines moved to " +
			"the untracked accounts. A category with categories under it cannot be deleted until they are. Deleting a " +
			"category that does not exist changes nothing. A deletion still running after a while is answered with " +
			"202 Accepted, and deleting again waits on it.",
		Tags:      tags,
		Errors:    []int{http.StatusConflict, http.StatusUnprocessableEntity},
		Responses: gateway.AccountDeletionResponses(api),
	}, func(ctx context.Context, in *DeleteCategoryInput) (*gateway.AccountDeletionOutput, error) {
		return gateway.DeleteAccount(ctx, c, workflows.DeleteCategoryWorkflowID(in.ID), categorieswf.DeleteCategory,
			ct.DeleteCategoryInput{LedgerAccountID: in.ID, ReplacementID: gateway.Replacement(in.ReplacementID)},
			func(result *lt.DeleteAccountResult) []lt.NegativeHolding { return result.Warnings })
	})
}
