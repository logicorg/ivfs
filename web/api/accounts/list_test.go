package accounts

import (
	"net/http"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestList checks that financial accounts are listed with their
// institution's name and closed state, and that the closed filter reaches
// ListFinancialAccounts.
func TestList(t *testing.T) {
	closedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	revolut := &model.Institution{ID: 1, Name: "Revolut"}
	personal := model.FinancialAccount{ID: 10, InstitutionID: 1, Name: "Revolut Personal", Institution: revolut}
	old := model.FinancialAccount{ID: 11, InstitutionID: 1, Name: "Revolut Old", Institution: revolut, ClosedAt: &closedAt}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"list the open accounts": {
			Method: http.MethodGet, Path: "/api/financial-accounts",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.ListFinancialAccounts, Params: lt.ListFinancialAccountsParams{},
				Result: lt.ListFinancialAccountsResult{FinancialAccounts: []model.FinancialAccount{personal}},
			},
			WantStatus: http.StatusOK,
			WantBody: `{"financialAccounts":[` +
				`{"id":10,"name":"Revolut Personal","institutionId":1,"institutionName":"Revolut","institutionClosed":false,"closed":false}]}`,
		},
		"list the closed accounts too": {
			Method: http.MethodGet, Path: "/api/financial-accounts?includeClosed=true",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.ListFinancialAccounts, Params: lt.ListFinancialAccountsParams{IncludeClosed: true},
				Result: lt.ListFinancialAccountsResult{FinancialAccounts: []model.FinancialAccount{personal, old}},
			},
			WantStatus: http.StatusOK,
			WantBody: `{"financialAccounts":[` +
				`{"id":10,"name":"Revolut Personal","institutionId":1,"institutionName":"Revolut","institutionClosed":false,"closed":false},` +
				`{"id":11,"name":"Revolut Old","institutionId":1,"institutionName":"Revolut","institutionClosed":false,"closed":true}]}`,
		},
		"list no accounts": {
			Method: http.MethodGet, Path: "/api/financial-accounts",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.ListFinancialAccounts, Params: lt.ListFinancialAccountsParams{},
				Result: lt.ListFinancialAccountsResult{},
			},
			WantStatus: http.StatusOK, WantBody: `{"financialAccounts":[]}`,
		},
	})
}
