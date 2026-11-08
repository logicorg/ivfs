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

func registerDelete(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-tag",
		Method:      http.MethodDelete,
		Path:        "/api/tags/{id}",
		Summary:     "Delete a tag",
		Description: "Deleting a tag that does not exist changes nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *DeleteTagInput) (*struct{}, error) {
		return nil, gateway.RunWorkflowWithoutResult(ctx, c, tagswf.DeleteTag, &tt.DeleteTagParams{ID: in.ID})
	})
}
