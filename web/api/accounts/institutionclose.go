package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerCloseInstitution(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "close-institution",
		Method:      http.MethodPost,
		Path:        "/api/institutions/{id}/close",
		Summary:     "Close an institution",
		Description: "An institution can be closed only once each of its financial accounts is. Closing a closed institution changes nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *CloseInput) (*struct{}, error) {
		return nil, gateway.RunWorkflowWithoutResult(ctx, c, institutionwf.CloseInstitution, it.CloseParams{FindBy: it.RetrieveInstitutionParams{ID: in.ID}})
	})
}
