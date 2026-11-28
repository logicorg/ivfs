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

// TestUpdateTransfer checks that the body and the path's id are passed to
// UpdateTransfer, with the amount that arrives and the charges when given,
// that the entry's id is returned, that a refusal keeps its status, and that a
// request Huma rejects runs no workflow.
func TestUpdateTransfer(t *testing.T) {
	const body = `{"fromLedgerAccountId":100,"toLedgerAccountId":104,"amount":"1200","occurredAt":"2026-10-03T10:00:00Z"`
	instalment := tt.UpdateTransferInput{
		JournalEntryID: 41, FromLedgerAccountID: 100, ToLedgerAccountID: 104,
		Amount: decimal.NewFromInt(1200), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		Charges: []tt.Split{{CategoryID: 220, Amount: decimal.NewFromInt(800)}},
	}
	toDollars := tt.UpdateTransferInput{
		JournalEntryID: 41, FromLedgerAccountID: 100, ToLedgerAccountID: 103,
		Amount: decimal.RequireFromString("300.5"), ToAmount: decimal.RequireFromString("324.15"),
		OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
	}
	update := func(params tt.UpdateTransferInput, err error) *apitest.WorkflowCall {
		call := &apitest.WorkflowCall{Workflow: transactionswf.UpdateTransfer, Params: params, Err: err}
		if err == nil {
			call.Result = tt.UpdateResult{JournalEntryID: 41}
		}
		return call
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"correct an entry to an instalment": {
			Method: http.MethodPut, Path: "/api/transactions/transfers/41", Body: body + `,"charges":[{"categoryId":220,"amount":"800"}]}`,
			Call:       update(instalment, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":41}`,
		},
		"into dollars": {
			Method: http.MethodPut, Path: "/api/transactions/transfers/41",
			Body:       `{"fromLedgerAccountId":100,"toLedgerAccountId":103,"amount":"300.50","toAmount":"324.15","occurredAt":"2026-10-03T10:00:00Z"}`,
			Call:       update(toDollars, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":41}`,
		},
		"two charges between currencies": {
			Method: http.MethodPut, Path: "/api/transactions/transfers/41", Body: body + `,"charges":[{"categoryId":220,"amount":"800"}]}`,
			Call: update(instalment, temporal.NewApplicationError(
				`"Cash" and "Principal" are in different currencies, so the transfer takes at most one charge, its exchange fee`, "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody: apitest.Problem(http.StatusUnprocessableEntity,
				`"Cash" and "Principal" are in different currencies, so the transfer takes at most one charge, its exchange fee`),
		},
		"a charge that is not a decimal": {
			Method: http.MethodPut, Path: "/api/transactions/transfers/41", Body: body + `,"charges":[{"categoryId":220,"amount":"eight"}]}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
