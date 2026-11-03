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

// TestRecordTransfer checks that the form is passed to RecordTransfer with its
// amounts exact (300.50 arrives as 300.5), and the amount that arrives and the
// note only when given, that the entry's id is returned, that a refusal keeps
// its status, and that a form Huma rejects runs no workflow.
func TestRecordTransfer(t *testing.T) {
	const form = `{"idempotencyKey":"3d4b","fromLedgerAccountId":100,"toLedgerAccountId":101,` +
		`"amount":"300.50","occurredAt":"2026-10-03T10:00:00Z"`
	cardPaidOff := tt.RecordTransferInput{
		IdempotencyKey: "3d4b", FromLedgerAccountID: 100, ToLedgerAccountID: 101,
		Amount: decimal.RequireFromString("300.5"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
	}
	withNote := cardPaidOff
	withNote.Description = "pay the card off"
	toDollars := tt.RecordTransferInput{
		IdempotencyKey: "3d4b", FromLedgerAccountID: 100, ToLedgerAccountID: 103,
		Amount: decimal.RequireFromString("300.5"), ToAmount: decimal.RequireFromString("324.15"),
		OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
	}
	record := func(params tt.RecordTransferInput, err error) *apitest.WorkflowCall {
		call := &apitest.WorkflowCall{Workflow: transactionswf.RecordTransfer, Params: params, Err: err}
		if err == nil {
			call.Result = tt.RecordResult{JournalEntryID: 44}
		}
		return call
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"record a transfer": {
			Method: http.MethodPost, Path: "/api/transactions/transfers", Body: form + `}`,
			Call:       record(cardPaidOff, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":44}`,
		},
		"with a note": {
			Method: http.MethodPost, Path: "/api/transactions/transfers", Body: form + `,"description":"pay the card off"}`,
			Call:       record(withNote, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":44}`,
		},
		"into another currency": {
			Method: http.MethodPost, Path: "/api/transactions/transfers",
			Body:       `{"idempotencyKey":"3d4b","fromLedgerAccountId":100,"toLedgerAccountId":103,"amount":"300.50","toAmount":"324.15","occurredAt":"2026-10-03T10:00:00Z"}`,
			Call:       record(toDollars, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":44}`,
		},
		"into another currency, with no amount that arrives": {
			Method: http.MethodPost, Path: "/api/transactions/transfers",
			Body: `{"idempotencyKey":"3d4b","fromLedgerAccountId":100,"toLedgerAccountId":103,"amount":"300.50","occurredAt":"2026-10-03T10:00:00Z"}`,
			Call: record(
				tt.RecordTransferInput{
					IdempotencyKey: "3d4b", FromLedgerAccountID: 100, ToLedgerAccountID: 103,
					Amount: decimal.RequireFromString("300.5"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
				},
				temporal.NewApplicationError(`"Cash" and "Cash (USD)" are in different currencies; give the amount that arrives as toAmount`, "ErrInvalidParameter"),
			),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody: apitest.Problem(http.StatusUnprocessableEntity,
				`"Cash" and "Cash (USD)" are in different currencies; give the amount that arrives as toAmount`),
		},
		"an unknown account": {
			Method: http.MethodPost, Path: "/api/transactions/transfers", Body: form + `}`,
			Call:       record(cardPaidOff, apitest.ChildWorkflowError("ErrNotFound", "resource LedgerAccount not found with ID: 101")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource LedgerAccount not found with ID: 101"),
		},
		"a key used by another entry": {
			Method: http.MethodPost, Path: "/api/transactions/transfers",
			Body: `{"idempotencyKey":"3d4b","fromLedgerAccountId":100,"toLedgerAccountId":103,"amount":"300.50","toAmount":"324.15","occurredAt":"2026-10-03T10:00:00Z"}`,
			Call: record(toDollars, apitest.ChildWorkflowError("ErrResourceConflict",
				"idempotency key belongs to an entry that is not an FX conversion")),
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "idempotency key belongs to an entry that is not an FX conversion"),
		},
		"no idempotency key": {
			Method: http.MethodPost, Path: "/api/transactions/transfers",
			Body:       `{"idempotencyKey":"","fromLedgerAccountId":100,"toLedgerAccountId":101,"amount":"300.50","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"source 0": {
			Method: http.MethodPost, Path: "/api/transactions/transfers",
			Body:       `{"idempotencyKey":"3d4b","fromLedgerAccountId":0,"toLedgerAccountId":101,"amount":"300.50","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"destination 0": {
			Method: http.MethodPost, Path: "/api/transactions/transfers",
			Body:       `{"idempotencyKey":"3d4b","fromLedgerAccountId":100,"toLedgerAccountId":0,"amount":"300.50","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"an amount that arrives that is not a decimal": {
			Method: http.MethodPost, Path: "/api/transactions/transfers",
			Body:       `{"idempotencyKey":"3d4b","fromLedgerAccountId":100,"toLedgerAccountId":103,"amount":"300.50","toAmount":"324,15","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"no occurrence date": {
			Method: http.MethodPost, Path: "/api/transactions/transfers",
			Body:       `{"idempotencyKey":"3d4b","fromLedgerAccountId":100,"toLedgerAccountId":101,"amount":"300.50"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
