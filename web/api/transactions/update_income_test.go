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

// TestUpdateIncome checks that the body and the path's id are passed to
// UpdateIncome, that the entry's id is returned, that a refusal keeps its
// status, and that a request Huma rejects runs no workflow.
func TestUpdateIncome(t *testing.T) {
	const body = `{"balanceLedgerAccountId":100,"categoryId":210,"amount":"2500","occurredAt":"2026-09-30T00:00:00Z"`
	salary := tt.UpdateIncomeInput{
		JournalEntryID: 41, BalanceLedgerAccountID: 100, CategoryID: 210,
		Amount: decimal.NewFromInt(2500), OccurredAt: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
	}
	withNote := salary
	withNote.Description, withNote.TagIDs = "September salary", []int64{3}
	update := func(params tt.UpdateIncomeInput, err error) *apitest.WorkflowCall {
		call := &apitest.WorkflowCall{Workflow: transactionswf.UpdateIncome, Params: params, Err: err}
		if err == nil {
			call.Result = tt.UpdateResult{JournalEntryID: 41}
		}
		return call
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"correct an entry to an income": {
			Method: http.MethodPut, Path: "/api/transactions/incomes/41", Body: body + `}`,
			Call:       update(salary, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":41}`,
		},
		"with a note and a tag": {
			Method: http.MethodPut, Path: "/api/transactions/incomes/41", Body: body + `,"description":"September salary","tagIds":[3]}`,
			Call:       update(withNote, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":41}`,
		},
		"a draft": {
			Method: http.MethodPut, Path: "/api/transactions/incomes/41", Body: body + `}`,
			Call: update(salary, apitest.ChildWorkflowError("ErrInvalidParameter",
				"journal entry 41 is not posted, and a replace would post it")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "journal entry 41 is not posted, and a replace would post it"),
		},
		"a closed account": {
			Method: http.MethodPut, Path: "/api/transactions/incomes/41", Body: body + `}`,
			Call:       update(salary, temporal.NewApplicationError("resource LedgerAccount is closed", "ErrResourceClosed")),
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource LedgerAccount is closed"),
		},
		"no category": {
			Method: http.MethodPut, Path: "/api/transactions/incomes/41",
			Body:       `{"balanceLedgerAccountId":100,"amount":"2500","occurredAt":"2026-09-30T00:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
