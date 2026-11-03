package journal

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerList(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "list-journal-entries",
		Method:      http.MethodGet,
		Path:        "/api/journal-entries",
		Summary:     "List a page of journal entries, by the date they occurred and then by id",
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *ListJournalEntriesInput) (*ListJournalEntriesOutput, error) {
		after, ok := decodeCursor(in.After)
		if !ok {
			return nil, huma.Error422UnprocessableEntity("after is not the next of a page",
				&huma.ErrorDetail{Location: "query.after", Value: in.After, Message: "not the next of a page"})
		}
		params := lt.ListJournalEntriesParams{
			LedgerAccountID: in.LedgerAccountID, CounterpartyID: in.CounterpartyID, From: in.From, To: in.To,
			Descending: in.Descending, Limit: in.Limit, After: after,
		}
		for _, status := range in.Statuses {
			params.Statuses = append(params.Statuses, model.JournalEntryStatus(status))
		}

		result, err := gateway.RunWorkflow[lt.ListJournalEntriesResult](ctx, c, journalwf.ListJournalEntries, params)
		if err != nil {
			return nil, err
		}
		out := &ListJournalEntriesOutput{Body: EntryPage{
			Entries: make([]Entry, 0, len(result.Entries)), Next: encodeCursor(result.Next),
		}}
		for n := range result.Entries {
			entry, err := newEntry(&result.Entries[n])
			if err != nil {
				return nil, gateway.WorkflowError(err)
			}
			out.Body.Entries = append(out.Body.Entries, entry)
		}
		return out, nil
	})
}
