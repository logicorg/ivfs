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

func registerOpen(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "open-account",
		Method:      http.MethodPost,
		Path:        "/api/accounts/open",
		Summary:     "Open a financial account with its balances",
		Description: "Finds or creates the account and gives it the balances its type needs in each currency, with their opening balances. " +
			"A missing institution is created only when createInstitution is set; otherwise the outcome is institution_not_found and nothing is written. " +
			"Opening an account that exists adds the currencies it does not hold yet.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *OpenAccountInput) (*OpenAccountOutput, error) {
		result, err := gateway.RunWorkflow[ast.OpenAccountResult](ctx, c, accountsetup.OpenAccount, newOpenAccountInput(in.Body))
		if err != nil {
			return nil, err
		}
		return &OpenAccountOutput{Body: OpenedAccount{
			Outcome:              string(result.Outcome),
			InstitutionID:        result.InstitutionID,
			InstitutionName:      result.InstitutionName,
			FinancialAccountID:   result.FinancialAccountID,
			FinancialAccountName: result.FinancialAccountName,
		}}, nil
	})
}

// newOpenAccountInput is form as OpenAccount takes it.
func newOpenAccountInput(form AccountForm) ast.OpenAccountInput {
	currencies := make([]ast.CurrencyBalance, 0, len(form.Currencies))
	for _, currency := range form.Currencies {
		currencies = append(currencies, ast.CurrencyBalance{Code: currency.Code, OpeningBalance: currency.OpeningBalance.Decimal})
	}
	return ast.OpenAccountInput{
		Type:              ast.AccountType(form.Type),
		InstitutionName:   form.InstitutionName,
		CreateInstitution: form.CreateInstitution,
		AccountName:       form.AccountName,
		Currencies:        currencies,
		OpeningBalancesAt: form.OpeningBalancesAt,
	}
}
