package journal

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerDelete(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-journal-entry",
		Method:      http.MethodDelete,
		Path:        "/api/journal-entries/{id}",
		Summary:     "Delete a journal entry with its postings, position movements, and reversals",
		Description: "The ledger is left as if the entry had never been recorded, and its idempotency key is free again. " +
			"Deleting an entry that does not exist changes nothing. A deletion that leaves a holding negative at some " +
			"point in time is made all the same, and the response warns about it.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *DeleteJournalEntryInput) (*DeleteJournalEntryOutput, error) {
		result, err := gateway.RunWorkflow[lt.DeleteJournalEntryResult](ctx, c, journalwf.DeleteJournalEntry,
			lt.DeleteJournalEntryInput{JournalEntryID: in.ID})
		if err != nil {
			return nil, err
		}
		return &DeleteJournalEntryOutput{Body: gateway.Deletion{Warnings: gateway.NewNegativeHoldings(result.Warnings)}}, nil
	})
}
