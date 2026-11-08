package tags

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	tagswf "github.com/logicorg/ivfs/finance/workflows/tags"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerUpdate(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "update-tag",
		Method:      http.MethodPatch,
		Path:        "/api/tags/{id}",
		Summary:     "Change a tag's name or description",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *UpdateTagInput) (*struct{}, error) {
		params := &tt.UpdateTagParams{ID: in.ID, Name: in.Body.Name, Description: in.Body.Description}
		return nil, gateway.RunWorkflowWithoutResult(ctx, c, tagswf.UpdateTag, params)
	})
}
