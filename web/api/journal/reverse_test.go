package journal

import (
	"net/http"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestReverse checks that reversing runs ReverseJournalEntry with no
// idempotency key, so the workflow picks one, returns the reversal's id and its
// warnings, and that its refusals keep their statuses.
func TestReverse(t *testing.T) {
	reverse := func(err error, result any) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: journalwf.ReverseJournalEntry,
			Params:   lt.ReverseJournalEntryInput{JournalEntryID: 7},
			Result:   result,
			Err:      err,
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"reverse an entry": {
			Method: http.MethodPost, Path: "/api/journal-entries/7/reverse",
			Call:       reverse(nil, lt.ReverseJournalEntryResult{JournalEntryID: 8}),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":8}`,
		},
		"reverse an entry that leaves a holding negative": {
			Method: http.MethodPost, Path: "/api/journal-entries/7/reverse",
			Call: reverse(nil, lt.ReverseJournalEntryResult{JournalEntryID: 8, Warnings: []lt.NegativeHolding{{
				FinancialAccountID: 3, FinancialAccountName: "IBKR", AssetID: 12, AssetName: "ACME",
				At: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), Units: decimal.NewFromInt(-90),
			}}}),
			WantStatus: http.StatusOK,
			WantBody: `{"journalEntryId":8,"warnings":[{"financialAccountId":3,"financialAccountName":"IBKR",` +
				`"assetId":12,"assetName":"ACME","at":"2026-03-03T00:00:00Z","units":"-90"}]}`,
		},
		"reverse an entry twice": {
			Method: http.MethodPost, Path: "/api/journal-entries/7/reverse",
			Call:       reverse(temporal.NewApplicationError("entry has already been reversed", "ErrResourceConflict"), nil),
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "entry has already been reversed"),
		},
		"reverse a draft": {
			Method: http.MethodPost, Path: "/api/journal-entries/7/reverse",
			Call:       reverse(temporal.NewApplicationError("only posted journal entries can be reversed", "ErrInvalidParameter"), nil),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "only posted journal entries can be reversed"),
		},
		"reverse an entry that does not exist": {
			Method: http.MethodPost, Path: "/api/journal-entries/7/reverse",
			Call:       reverse(temporal.NewApplicationError("resource JournalEntry not found with ID: 7", "ErrNotFound"), nil),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource JournalEntry not found with ID: 7"),
		},
		"an id that is not one": {
			Method: http.MethodPost, Path: "/api/journal-entries/0/reverse", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
