package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerDeleteInstitution(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-institution",
		Method:      http.MethodDelete,
		Path:        "/api/institutions/{id}",
		Summary:     "Delete an institution with its financial accounts",
		Description: "Each financial account is deleted as DELETE /api/financial-accounts/{id} deletes it. Deleting an " +
			"institution that does not exist changes nothing. A deletion still running after a while is answered with " +
			"202 Accepted, and deleting again waits on it.",
		Tags:      tags,
		Errors:    []int{http.StatusConflict, http.StatusUnprocessableEntity},
		Responses: gateway.AccountDeletionResponses(api),
	}, func(ctx context.Context, in *DeleteInstitutionInput) (*gateway.AccountDeletionOutput, error) {
		return gateway.DeleteAccount(ctx, c, workflows.DeleteInstitutionWorkflowID(in.ID), institutionwf.DeleteInstitution,
			it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: in.ID}},
			func(*struct{}) []lt.NegativeHolding { return nil })
	})
}
