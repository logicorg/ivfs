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

func registerBalances(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "get-account-balances",
		Method:      http.MethodGet,
		Path:        "/api/financial-accounts/{id}/balances",
		Summary:     "Get the balance of each of a financial account's ledger accounts",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, in *AccountBalancesInput) (*AccountBalancesOutput, error) {
		result, err := gateway.RunWorkflow[lt.FinancialAccountBalancesResult](ctx, c, reportingwf.FinancialAccountBalances,
			lt.FinancialAccountBalancesParams{FinancialAccountID: in.ID, At: in.At})
		if err != nil {
			return nil, err
		}
		out := &AccountBalancesOutput{
			Body: AccountBalances{FinancialAccountID: result.FinancialAccountID, Balances: make([]Balance, 0, len(result.Balances))},
		}
		for _, balance := range result.Balances {
			out.Body.Balances = append(out.Body.Balances, newBalance(balance))
		}
		return out, nil
	})
}
