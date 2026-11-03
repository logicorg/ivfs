package categories

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerTotals(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "get-category-totals",
		Method:      http.MethodGet,
		Path:        "/api/categories/totals",
		Summary:     "Total what each category earned or spent over a period",
		Description: "The period is half-open: an entry that occurred at its start counts, one at its end does not, so consecutive periods neither overlap nor leave a gap. Only posted entries count.",
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *CategoryTotalsInput) (*CategoryTotalsOutput, error) {
		result, err := gateway.RunWorkflow[lt.CategoryTotalsResult](ctx, c, reportingwf.CategoryTotals,
			lt.CategoryTotalsParams{From: in.From, To: in.To})
		if err != nil {
			return nil, err
		}
		out := &CategoryTotalsOutput{Body: CategoryTotalList{Totals: make([]CategoryTotal, 0, len(result.Totals))}}
		for _, total := range result.Totals {
			out.Body.Totals = append(out.Body.Totals, CategoryTotal{
				ID: total.LedgerAccountID, Kind: string(total.Kind),
				CurrencyID: total.CurrencyID, CurrencyCode: total.CurrencyCode, Total: gateway.NewAmount(total.Total),
			})
		}
		return out, nil
	})
}
