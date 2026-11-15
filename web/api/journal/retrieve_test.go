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

// TestRetrieve checks that an entry's detail has its counterparty's name and
// its dates besides what the list shows, that an entry without them leaves
// them out, that a posting has its category when it carries one, and that a
// missing entry is a 404.
func TestRetrieve(t *testing.T) {
	day := time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC)
	cash := &model.LedgerAccount{ID: 100, Name: "Cash", Kind: model.LedgerAccountAsset, CurrencyID: 1}
	salary := &model.LedgerAccount{ID: 300, Name: "Salary", Kind: model.LedgerAccountIncome, CurrencyID: 1}
	postings := []model.Posting{
		{ID: 70, LineNumber: 1, LedgerAccountID: 100, LedgerAccount: cash, Amount: decimal.RequireFromString("2500"), Side: model.PostingDebit},
		{
			ID: 71, LineNumber: 2, LedgerAccountID: 300, LedgerAccount: salary, Amount: decimal.RequireFromString("2500"), Side: model.PostingCredit,
			CategoryID: ptr.Int64(40), Category: &model.Category{ID: 40, Name: "Wages", Kind: model.LedgerAccountIncome},
		},
	}
	postingsBody := `"postings":[` +
		`{"lineNumber":1,"ledgerAccountId":100,"ledgerAccountName":"Cash","kind":"asset","currencyId":1,"amount":"2500","side":"debit"},` +
		`{"lineNumber":2,"ledgerAccountId":300,"ledgerAccountName":"Salary","kind":"income","currencyId":1,"amount":"2500","side":"credit","categoryId":40,"categoryName":"Wages"}]`

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"an entry with a counterparty": {
			Method: http.MethodGet, Path: "/api/journal-entries/7",
			Call: &apitest.WorkflowCall{
				Workflow: journalwf.RetrieveJournalEntry,
				Params:   lt.RetrieveJournalEntryParams{ID: 7},
				Result: lt.RetrieveJournalEntryResult{JournalEntry: &model.JournalEntry{
					ID: 7, OccurredAt: day, Description: ptr.NewPtr("January salary"), Status: model.JournalEntryPosted,
					CounterpartyID: ptr.Int64(5), Counterparty: &model.Counterparty{ID: 5, Name: "Acme"},
					PostedAt: ptr.NewPtr(day.Add(time.Hour)), CreatedAt: day.Add(time.Hour), Postings: postings,
				}},
			},
			WantStatus: http.StatusOK,
			WantBody: `{"id":7,"occurredAt":"2026-01-31T10:00:00Z","description":"January salary","status":"posted",` + postingsBody +
				`,"counterpartyId":5,"counterpartyName":"Acme","postedAt":"2026-01-31T11:00:00Z","createdAt":"2026-01-31T11:00:00Z"}`,
		},
		"a draft without a counterparty": {
			Method: http.MethodGet, Path: "/api/journal-entries/8",
			Call: &apitest.WorkflowCall{
				Workflow: journalwf.RetrieveJournalEntry,
				Params:   lt.RetrieveJournalEntryParams{ID: 8},
				Result: lt.RetrieveJournalEntryResult{JournalEntry: &model.JournalEntry{
					ID: 8, OccurredAt: day, Status: model.JournalEntryDraft, CreatedAt: day, Postings: postings,
				}},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"id":8,"occurredAt":"2026-01-31T10:00:00Z","status":"draft",` + postingsBody + `,"createdAt":"2026-01-31T10:00:00Z"}`,
		},
		"an entry that does not exist": {
			Method: http.MethodGet, Path: "/api/journal-entries/99",
			Call: &apitest.WorkflowCall{
				Workflow: journalwf.RetrieveJournalEntry,
				Params:   lt.RetrieveJournalEntryParams{ID: 99},
				Err:      temporal.NewApplicationError("resource JournalEntry not found with ID: 99", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource JournalEntry not found with ID: 99"),
		},
		"an id that is not one": {
			Method: http.MethodGet, Path: "/api/journal-entries/0", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
