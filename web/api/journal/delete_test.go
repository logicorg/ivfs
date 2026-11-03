package journal

import (
	"net/http"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
)

// TestDelete checks that deleting runs DeleteJournalEntry with the id, returns
// its warnings, and that an invalid id runs no workflow.
func TestDelete(t *testing.T) {
	remove := func(result lt.DeleteJournalEntryResult) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: journalwf.DeleteJournalEntry,
			Params:   lt.DeleteJournalEntryInput{JournalEntryID: 7},
			Result:   result,
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"delete an entry": {
			Method: http.MethodDelete, Path: "/api/journal-entries/7",
			Call:       remove(lt.DeleteJournalEntryResult{}),
			WantStatus: http.StatusOK, WantBody: `{}`,
		},
		"delete an entry that leaves a holding negative": {
			Method: http.MethodDelete, Path: "/api/journal-entries/7",
			Call: remove(lt.DeleteJournalEntryResult{Warnings: []lt.NegativeHolding{{
				FinancialAccountID: 3, FinancialAccountName: "IBKR", AssetID: 12, AssetName: "ACME",
				At: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), Units: decimal.NewFromInt(-10),
			}}}),
			WantStatus: http.StatusOK,
			WantBody: `{"warnings":[{"financialAccountId":3,"financialAccountName":"IBKR",` +
				`"assetId":12,"assetName":"ACME","at":"2026-03-03T00:00:00Z","units":"-10"}]}`,
		},
		"an id that is not one": {
			Method: http.MethodDelete, Path: "/api/journal-entries/0", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
