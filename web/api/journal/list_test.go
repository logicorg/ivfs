package journal

import (
	"net/http"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestList checks that each filter and the cursor reach ListJournalEntries,
// that the next page's cursor goes back as after, that each entry has its
// postings with their accounts' names and their categories, and that a cursor
// the API did not return, or an entry without its postings' accounts or
// categories, is refused.
func TestList(t *testing.T) {
	day := time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC)
	cash := &model.LedgerAccount{ID: 100, Name: "Cash", Kind: model.LedgerAccountAsset, CurrencyID: 1}
	groceries := &model.LedgerAccount{ID: 200, Name: "Groceries", Kind: model.LedgerAccountExpense, CurrencyID: 1}
	shopping := model.JournalEntry{
		ID: 7, OccurredAt: day, Description: ptr.NewPtr("Supermarket"), ExternalID: ptr.NewPtr("bank-42"),
		Status: model.JournalEntryPosted,
		Postings: []model.Posting{
			{
				ID: 70, LineNumber: 1, LedgerAccountID: 200, LedgerAccount: groceries, Amount: decimal.RequireFromString("12345678901234.56"), Side: model.PostingDebit,
				CategoryID: ptr.Int64(40), Category: &model.Category{ID: 40, Name: "Food", Kind: model.LedgerAccountExpense},
			},
			{ID: 71, LineNumber: 2, LedgerAccountID: 100, LedgerAccount: cash, Amount: decimal.RequireFromString("12345678901234.56"), Side: model.PostingCredit},
		},
	}
	refund := model.JournalEntry{
		ID: 8, OccurredAt: day.Add(time.Hour), Status: model.JournalEntryPosted,
		Postings: []model.Posting{
			{ID: 80, LineNumber: 1, LedgerAccountID: 200, LedgerAccount: groceries, Amount: decimal.RequireFromString("1.5"), Side: model.PostingCredit},
			{ID: 81, LineNumber: 2, LedgerAccountID: 100, LedgerAccount: cash, Amount: decimal.RequireFromString("1.5"), Side: model.PostingDebit},
		},
	}
	shoppingBody := `{"id":7,"occurredAt":"2026-01-31T10:00:00Z","description":"Supermarket","externalId":"bank-42","status":"posted","postings":[` +
		`{"lineNumber":1,"ledgerAccountId":200,"ledgerAccountName":"Groceries","kind":"expense","currencyId":1,"amount":"12345678901234.56","side":"debit","categoryId":40,"categoryName":"Food"},` +
		`{"lineNumber":2,"ledgerAccountId":100,"ledgerAccountName":"Cash","kind":"asset","currencyId":1,"amount":"12345678901234.56","side":"credit"}]}`
	refundBody := `{"id":8,"occurredAt":"2026-01-31T11:00:00Z","status":"posted","postings":[` +
		`{"lineNumber":1,"ledgerAccountId":200,"ledgerAccountName":"Groceries","kind":"expense","currencyId":1,"amount":"1.5","side":"credit"},` +
		`{"lineNumber":2,"ledgerAccountId":100,"ledgerAccountName":"Cash","kind":"asset","currencyId":1,"amount":"1.5","side":"debit"}]}`

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"an account's posted entries, newest first": {
			Method: http.MethodGet, Path: "/api/journal-entries?ledgerAccountId=100&status=posted&descending=true&limit=2",
			Call: &apitest.WorkflowCall{
				Workflow: journalwf.ListJournalEntries,
				Params: lt.ListJournalEntriesParams{
					LedgerAccountID: 100, Statuses: []model.JournalEntryStatus{model.JournalEntryPosted}, Descending: true, Limit: 2,
				},
				Result: lt.ListJournalEntriesResult{
					Entries: []model.JournalEntry{refund, shopping}, Next: &lt.JournalEntryCursor{OccurredAt: day, ID: 7},
				},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"entries":[` + refundBody + `,` + shoppingBody + `],"next":"2026-01-31T10:00:00Z_7"}`,
		},
		"the next page": {
			Method: http.MethodGet, Path: "/api/journal-entries?ledgerAccountId=100&after=2026-01-31T10:00:00Z_7",
			Call: &apitest.WorkflowCall{
				Workflow: journalwf.ListJournalEntries,
				Params: lt.ListJournalEntriesParams{
					LedgerAccountID: 100, After: &lt.JournalEntryCursor{OccurredAt: day, ID: 7},
				},
				Result: lt.ListJournalEntriesResult{Entries: []model.JournalEntry{shopping}},
			},
			WantStatus: http.StatusOK, WantBody: `{"entries":[` + shoppingBody + `]}`,
		},
		"every entry, in a date range and several states": {
			Method: http.MethodGet,
			Path:   "/api/journal-entries?from=2026-01-01T00:00:00Z&to=2026-02-01T00:00:00Z&status=draft&status=posted",
			Call: &apitest.WorkflowCall{
				Workflow: journalwf.ListJournalEntries,
				Params: lt.ListJournalEntriesParams{
					Statuses: []model.JournalEntryStatus{model.JournalEntryDraft, model.JournalEntryPosted},
					From:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				},
				Result: lt.ListJournalEntriesResult{},
			},
			WantStatus: http.StatusOK, WantBody: `{"entries":[]}`,
		},
		"a counterparty's entries": {
			Method: http.MethodGet, Path: "/api/journal-entries?counterpartyId=5",
			Call: &apitest.WorkflowCall{
				Workflow: journalwf.ListJournalEntries,
				Params:   lt.ListJournalEntriesParams{CounterpartyID: 5},
				Result:   lt.ListJournalEntriesResult{},
			},
			WantStatus: http.StatusOK, WantBody: `{"entries":[]}`,
		},
		"a range the workflow refuses": {
			Method: http.MethodGet, Path: "/api/journal-entries?from=2026-02-01T00:00:00Z&to=2026-01-01T00:00:00Z",
			Call: &apitest.WorkflowCall{
				Workflow: journalwf.ListJournalEntries,
				Params: lt.ListJournalEntriesParams{
					From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				Err: temporal.NewApplicationError("the end of the date range must be after its start", "ErrInvalidParameter"),
			},
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "the end of the date range must be after its start"),
		},
		"an entry without its postings' accounts": {
			Method: http.MethodGet, Path: "/api/journal-entries",
			Call: &apitest.WorkflowCall{
				Workflow: journalwf.ListJournalEntries,
				Params:   lt.ListJournalEntriesParams{},
				Result: lt.ListJournalEntriesResult{Entries: []model.JournalEntry{{
					ID: 9, OccurredAt: day, Status: model.JournalEntryPosted,
					Postings: []model.Posting{{ID: 90, LineNumber: 1, LedgerAccountID: 100, Side: model.PostingDebit}},
				}}},
			},
			WantStatus: http.StatusInternalServerError,
		},
		"an entry without its posting's category": {
			Method: http.MethodGet, Path: "/api/journal-entries",
			Call: &apitest.WorkflowCall{
				Workflow: journalwf.ListJournalEntries,
				Params:   lt.ListJournalEntriesParams{},
				Result: lt.ListJournalEntriesResult{Entries: []model.JournalEntry{{
					ID: 9, OccurredAt: day, Status: model.JournalEntryPosted,
					Postings: []model.Posting{{ID: 90, LineNumber: 1, LedgerAccountID: 200, LedgerAccount: groceries, Side: model.PostingDebit, CategoryID: ptr.Int64(40)}},
				}}},
			},
			WantStatus: http.StatusInternalServerError,
		},
		"an after the API did not return": {
			Method: http.MethodGet, Path: "/api/journal-entries?after=yesterday", WantStatus: http.StatusUnprocessableEntity,
		},
		"a state that does not exist": {
			Method: http.MethodGet, Path: "/api/journal-entries?status=archived", WantStatus: http.StatusUnprocessableEntity,
		},
		"a page that is too large": {
			Method: http.MethodGet, Path: "/api/journal-entries?limit=501", WantStatus: http.StatusUnprocessableEntity,
		},
		"an account id that is not one": {
			Method: http.MethodGet, Path: "/api/journal-entries?ledgerAccountId=0", WantStatus: http.StatusUnprocessableEntity,
		},
		"a counterparty id that is not one": {
			Method: http.MethodGet, Path: "/api/journal-entries?counterpartyId=0", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
