package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	"github.com/logicorg/ivfs/finance/workflows/facade/accountsetup"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerAddBalance(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "add-balance",
		Method:      http.MethodPost,
		Path:        "/api/financial-accounts/{id}/balances",
		Summary:     "Add a named balance to a financial account",
		Description: "Adds an asset balance, such as a savings vault, in a currency, and books its opening balance. " +
			"A balance with the same name and currency in the account is returned unchanged, with no second opening entry.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *AddBalanceInput) (*AddBalanceOutput, error) {
		result, err := gateway.RunWorkflow[ast.AddBalanceResult](ctx, c, accountsetup.AddBalance, ast.AddBalanceInput{
			FinancialAccountID: in.ID,
			Name:               in.Body.Name,
			CurrencyCode:       in.Body.CurrencyCode,
			OpeningBalance:     in.Body.OpeningBalance.Decimal,
			OpeningBalanceAt:   in.Body.OpeningBalanceAt,
		})
		if err != nil {
			return nil, err
		}
		added := AddedBalance{LedgerAccountID: result.LedgerAccountID}
		for _, balance := range result.Balances.Balances {
			if balance.LedgerAccountID == result.LedgerAccountID {
				added.LedgerAccountName = balance.LedgerAccountName
			}
		}
		return &AddBalanceOutput{Body: added}, nil
	})
}
