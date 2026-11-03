package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerLedgerBalance(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "get-ledger-account-balance",
		Method:      http.MethodGet,
		Path:        "/api/ledger-accounts/{id}/balance",
		Summary:     "Get the balance of a ledger account, with its name, kind, and currency",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, in *LedgerAccountBalanceInput) (*LedgerAccountBalanceOutput, error) {
		result, err := gateway.RunWorkflow[lt.LedgerAccountBalanceResult](ctx, c, reportingwf.LedgerAccountBalance,
			lt.LedgerAccountBalanceParams{LedgerAccountID: in.ID, At: in.At})
		if err != nil {
			return nil, err
		}
		return &LedgerAccountBalanceOutput{Body: newBalance(*result)}, nil
	})
}
