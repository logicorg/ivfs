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

// TestRecordIncome checks that the form is passed to RecordIncome with its
// amount exact (2500.00 arrives as 2500) and its note only when given, that
// the entry's id is returned, that a refusal keeps its status, and that a form
// Huma rejects runs no workflow.
func TestRecordIncome(t *testing.T) {
	const form = `{"idempotencyKey":"7a2f","balanceLedgerAccountId":100,"categoryLedgerAccountId":210,` +
		`"amount":"2500.00","occurredAt":"2026-09-30T09:00:00Z"`
	salary := tt.RecordIncomeInput{
		IdempotencyKey: "7a2f", BalanceLedgerAccountID: 100, CategoryLedgerAccountID: 210,
		Amount: decimal.RequireFromString("2500"), OccurredAt: time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC),
	}
	withNote := salary
	withNote.Description = "September salary"
	record := func(params tt.RecordIncomeInput, err error) *apitest.WorkflowCall {
		call := &apitest.WorkflowCall{Workflow: transactionswf.RecordIncome, Params: params, Err: err}
		if err == nil {
			call.Result = tt.RecordResult{JournalEntryID: 43}
		}
		return call
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"record an income": {
			Method: http.MethodPost, Path: "/api/transactions/incomes", Body: form + `}`,
			Call:       record(salary, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":43}`,
		},
		"with a note": {
			Method: http.MethodPost, Path: "/api/transactions/incomes", Body: form + `,"description":"September salary"}`,
			Call:       record(withNote, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":43}`,
		},
		"a negative amount": {
			Method: http.MethodPost, Path: "/api/transactions/incomes",
			Body: `{"idempotencyKey":"7a2f","balanceLedgerAccountId":100,"categoryLedgerAccountId":210,"amount":"-2500.00","occurredAt":"2026-09-30T09:00:00Z"}`,
			Call: record(
				tt.RecordIncomeInput{
					IdempotencyKey: "7a2f", BalanceLedgerAccountID: 100, CategoryLedgerAccountID: 210,
					Amount: decimal.RequireFromString("-2500"), OccurredAt: time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC),
				},
				temporal.NewApplicationError("validation error: amount must be greater than zero", "ValidationError"),
			),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "validation error: amount must be greater than zero"),
		},
		"an unknown account": {
			Method: http.MethodPost, Path: "/api/transactions/incomes", Body: form + `}`,
			Call:       record(salary, apitest.ChildWorkflowError("ErrNotFound", "resource LedgerAccount not found with ID: 210")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource LedgerAccount not found with ID: 210"),
		},
		"a closed account": {
			Method: http.MethodPost, Path: "/api/transactions/incomes", Body: form + `}`,
			Call:       record(salary, apitest.ChildWorkflowError("ErrResourceClosed", "resource LedgerAccount is closed")),
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource LedgerAccount is closed"),
		},
		"no idempotency key": {
			Method: http.MethodPost, Path: "/api/transactions/incomes",
			Body:       `{"idempotencyKey":"","balanceLedgerAccountId":100,"categoryLedgerAccountId":210,"amount":"2500.00","occurredAt":"2026-09-30T09:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"balance 0": {
			Method: http.MethodPost, Path: "/api/transactions/incomes",
			Body:       `{"idempotencyKey":"7a2f","balanceLedgerAccountId":0,"categoryLedgerAccountId":210,"amount":"2500.00","occurredAt":"2026-09-30T09:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"category 0": {
			Method: http.MethodPost, Path: "/api/transactions/incomes",
			Body:       `{"idempotencyKey":"7a2f","balanceLedgerAccountId":100,"categoryLedgerAccountId":0,"amount":"2500.00","occurredAt":"2026-09-30T09:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"no amount": {
			Method: http.MethodPost, Path: "/api/transactions/incomes",
			Body:       `{"idempotencyKey":"7a2f","balanceLedgerAccountId":100,"categoryLedgerAccountId":210,"occurredAt":"2026-09-30T09:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"a time that is not RFC 3339": {
			Method: http.MethodPost, Path: "/api/transactions/incomes",
			Body:       `{"idempotencyKey":"7a2f","balanceLedgerAccountId":100,"categoryLedgerAccountId":210,"amount":"2500.00","occurredAt":"30/09/2026"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
