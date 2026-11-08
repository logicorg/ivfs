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

func registerList(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "list-tags",
		Method:      http.MethodGet,
		Path:        "/api/tags",
		Summary:     "List the tags, sorted by name",
		Tags:        tags,
	}, func(ctx context.Context, _ *ListTagsInput) (*ListTagsOutput, error) {
		result, err := gateway.RunWorkflow[tt.ListTagsResult](ctx, c, tagswf.ListTags, &tt.ListTagsParams{})
		if err != nil {
			return nil, err
		}
		out := &ListTagsOutput{Body: TagList{Tags: make([]Tag, 0, len(result.Tags))}}
		for _, tag := range result.Tags {
			out.Body.Tags = append(out.Body.Tags, Tag{ID: tag.ID, Name: tag.Name, Description: tag.Description})
		}
		return out, nil
	})
}
