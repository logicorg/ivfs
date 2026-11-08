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

func registerCreate(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "create-tag",
		Method:      http.MethodPost,
		Path:        "/api/tags",
		Summary:     "Create a tag, or find the one with its name",
		Description: "Names are compared ignoring case and surrounding space. A tag that exists is returned unchanged, whatever its description.",
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *CreateTagInput) (*CreateTagOutput, error) {
		result, err := gateway.RunWorkflow[tt.InsertTagResult](ctx, c, tagswf.InsertTag,
			tt.InsertTagParams{Name: in.Body.Name, Description: in.Body.Description})
		if err != nil {
			return nil, err
		}
		return &CreateTagOutput{Body: CreatedTag{ID: result.ID}}, nil
	})
}
