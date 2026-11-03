package health

import (
	"context"
	"log"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

func registerGet(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        "/api/health",
		Summary:     "Check that the gateway can reach Temporal",
		Tags:        []string{"health"},
		Errors:      []int{http.StatusServiceUnavailable},
	}, func(ctx context.Context, _ *struct{}) (*Output, error) {
		if _, err := c.CheckHealth(ctx, &client.CheckHealthRequest{}); err != nil {
			log.Printf("health check: %v", err)
			return nil, huma.Error503ServiceUnavailable("Temporal is unreachable")
		}
		return &Output{Body: Health{Status: "ok"}}, nil
	})
}
