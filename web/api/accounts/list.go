package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerList(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "list-financial-accounts",
		Method:      http.MethodGet,
		Path:        "/api/financial-accounts",
		Summary:     "List the financial accounts, with their institutions",
		Tags:        tags,
	}, func(ctx context.Context, in *ListFinancialAccountsInput) (*ListFinancialAccountsOutput, error) {
		result, err := gateway.RunWorkflow[lt.ListFinancialAccountsResult](ctx, c, ledgeraccountswf.ListFinancialAccounts,
			lt.ListFinancialAccountsParams{IncludeClosed: in.IncludeClosed})
		if err != nil {
			return nil, err
		}
		out := &ListFinancialAccountsOutput{
			Body: FinancialAccountList{FinancialAccounts: make([]FinancialAccount, 0, len(result.FinancialAccounts))},
		}
		for _, account := range result.FinancialAccounts {
			summary := FinancialAccount{
				ID: account.ID, Name: account.Name, InstitutionID: account.InstitutionID, Closed: account.ClosedAt != nil,
			}
			if account.Institution != nil {
				summary.InstitutionName = account.Institution.Name
				summary.InstitutionClosed = account.Institution.ClosedAt != nil
			}
			out.Body.FinancialAccounts = append(out.Body.FinancialAccounts, summary)
		}
		return out, nil
	})
}
