package transactions

import (
	"net/http"
	"testing"
	"time"

	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	transactionswf "github.com/logicorg/ivfs/finance/workflows/facade/transactions"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestRecordExpense checks that the form is passed to RecordExpense with its
// amount exact (12.50 arrives as 12.5) and its note only when given, that the
// entry's id is returned, that a refusal keeps its status, and that a form
// Huma rejects runs no workflow.
func TestRecordExpense(t *testing.T) {
	const form = `{"idempotencyKey":"9c1e","balanceLedgerAccountId":100,"categoryLedgerAccountId":200,` +
		`"amount":"12.50","occurredAt":"2026-10-03T10:00:00Z"`
	groceries := tt.RecordExpenseInput{
		IdempotencyKey: "9c1e", BalanceLedgerAccountID: 100, CategoryLedgerAccountID: 200,
		Amount: decimal.RequireFromString("12.5"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
	}
	withNote := groceries
	withNote.Description = "weekly shop"
	record := func(params tt.RecordExpenseInput, err error) *apitest.WorkflowCall {
		call := &apitest.WorkflowCall{Workflow: transactionswf.RecordExpense, Params: params, Err: err}
		if err == nil {
			call.Result = tt.RecordResult{JournalEntryID: 42}
		}
		return call
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"record an expense": {
			Method: http.MethodPost, Path: "/api/transactions/expenses", Body: form + `}`,
			Call:       record(groceries, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":42}`,
		},
		"with a note": {
			Method: http.MethodPost, Path: "/api/transactions/expenses", Body: form + `,"description":"weekly shop"}`,
			Call:       record(withNote, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":42}`,
		},
		"a category of the wrong kind": {
			Method: http.MethodPost, Path: "/api/transactions/expenses", Body: form + `}`,
			Call: record(groceries, temporal.NewApplicationError(
				`"Salary" is an income category, not an expense one`, "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, `"Salary" is an income category, not an expense one`),
		},
		"an unknown account": {
			Method: http.MethodPost, Path: "/api/transactions/expenses", Body: form + `}`,
			Call:       record(groceries, apitest.ChildWorkflowError("ErrNotFound", "resource LedgerAccount not found with ID: 100")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource LedgerAccount not found with ID: 100"),
		},
		"a closed account": {
			Method: http.MethodPost, Path: "/api/transactions/expenses", Body: form + `}`,
			Call:       record(groceries, apitest.ChildWorkflowError("ErrResourceClosed", "resource LedgerAccount is closed")),
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource LedgerAccount is closed"),
		},
		"no idempotency key": {
			Method: http.MethodPost, Path: "/api/transactions/expenses",
			Body:       `{"idempotencyKey":"","balanceLedgerAccountId":100,"categoryLedgerAccountId":200,"amount":"12.50","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"balance 0": {
			Method: http.MethodPost, Path: "/api/transactions/expenses",
			Body:       `{"idempotencyKey":"9c1e","balanceLedgerAccountId":0,"categoryLedgerAccountId":200,"amount":"12.50","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"category 0": {
			Method: http.MethodPost, Path: "/api/transactions/expenses",
			Body:       `{"idempotencyKey":"9c1e","balanceLedgerAccountId":100,"categoryLedgerAccountId":0,"amount":"12.50","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"an amount that is not a decimal": {
			Method: http.MethodPost, Path: "/api/transactions/expenses",
			Body:       `{"idempotencyKey":"9c1e","balanceLedgerAccountId":100,"categoryLedgerAccountId":200,"amount":"1,200","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"no occurrence date": {
			Method: http.MethodPost, Path: "/api/transactions/expenses",
			Body:       `{"idempotencyKey":"9c1e","balanceLedgerAccountId":100,"categoryLedgerAccountId":200,"amount":"12.50"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
