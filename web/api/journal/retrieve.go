package journal

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerRetrieve(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "get-journal-entry",
		Method:      http.MethodGet,
		Path:        "/api/journal-entries/{id}",
		Summary:     "Get a journal entry with its postings and counterparty",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, in *RetrieveJournalEntryInput) (*RetrieveJournalEntryOutput, error) {
		result, err := gateway.RunWorkflow[lt.RetrieveJournalEntryResult](ctx, c, journalwf.RetrieveJournalEntry,
			lt.RetrieveJournalEntryParams{ID: in.ID})
		if err != nil {
			return nil, err
		}
		if result.JournalEntry == nil {
			return nil, gateway.WorkflowError(fmt.Errorf("journal entry %d was retrieved empty", in.ID))
		}
		entry, err := newEntry(result.JournalEntry)
		if err != nil {
			return nil, gateway.WorkflowError(err)
		}

		detail := EntryDetail{
			Entry: entry, CounterpartyID: result.JournalEntry.CounterpartyID,
			PostedAt: result.JournalEntry.PostedAt, CreatedAt: result.JournalEntry.CreatedAt,
		}
		if result.JournalEntry.Counterparty != nil {
			detail.CounterpartyName = result.JournalEntry.Counterparty.Name
		}
		return &RetrieveJournalEntryOutput{Body: detail}, nil
	})
}
