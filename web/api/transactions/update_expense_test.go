package transactions

import (
	"net/http"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	transactionswf "github.com/logicorg/ivfs/finance/workflows/facade/transactions"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
)

// TestUpdateExpense checks that the body and the path's id are passed to
// UpdateExpense, with lines when given, that the entry's id and the update's
// warnings are returned, that a refusal keeps its status, and that a request
// Huma rejects runs no workflow.
func TestUpdateExpense(t *testing.T) {
	const body = `{"balanceLedgerAccountId":100,"amount":"12.50","occurredAt":"2026-10-03T10:00:00Z"`
	groceries := tt.UpdateExpenseInput{
		JournalEntryID: 41, BalanceLedgerAccountID: 100, CategoryID: 200,
		Amount: decimal.RequireFromString("12.5"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
	}
	withLines := groceries
	withLines.CategoryID, withLines.Lines = 0, []tt.Split{
		{CategoryID: 200, Amount: decimal.NewFromInt(10)}, {CategoryID: 201, Amount: decimal.RequireFromString("2.5")},
	}
	update := func(params tt.UpdateExpenseInput, result tt.UpdateResult, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{Workflow: transactionswf.UpdateExpense, Params: params, Result: result, Err: err}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"correct an entry to an expense": {
			Method: http.MethodPut, Path: "/api/transactions/expenses/41", Body: body + `,"categoryId":200}`,
			Call:       update(groceries, tt.UpdateResult{JournalEntryID: 41}, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":41}`,
		},
		"with lines": {
			Method: http.MethodPut, Path: "/api/transactions/expenses/41",
			Body:       body + `,"lines":[{"categoryId":200,"amount":"10"},{"categoryId":201,"amount":"2.50"}]}`,
			Call:       update(withLines, tt.UpdateResult{JournalEntryID: 41}, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":41}`,
		},
		"an entry that was a trade": {
			Method: http.MethodPut, Path: "/api/transactions/expenses/41", Body: body + `,"categoryId":200}`,
			Call: update(groceries, tt.UpdateResult{JournalEntryID: 41, Warnings: []lt.NegativeHolding{{
				FinancialAccountID: 3, FinancialAccountName: "IBKR", AssetID: 12, AssetName: "ACME",
				At: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), Units: decimal.NewFromInt(-10),
			}}}, nil),
			WantStatus: http.StatusOK,
			WantBody: `{"journalEntryId":41,"warnings":[{"financialAccountId":3,"financialAccountName":"IBKR",` +
				`"assetId":12,"assetName":"ACME","at":"2026-03-03T00:00:00Z","units":"-10"}]}`,
		},
		"an unknown entry": {
			Method: http.MethodPut, Path: "/api/transactions/expenses/41", Body: body + `,"categoryId":200}`,
			Call: update(groceries, tt.UpdateResult{},
				apitest.ChildWorkflowError("ErrNotFound", "resource JournalEntry not found with ID: 41")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource JournalEntry not found with ID: 41"),
		},
		"an id that is not one": {
			Method: http.MethodPut, Path: "/api/transactions/expenses/0", Body: body + `,"categoryId":200}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"an idempotency key, which the entry keeps its own of": {
			Method: http.MethodPut, Path: "/api/transactions/expenses/41", Body: body + `,"categoryId":200,"idempotencyKey":"9c1e"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
