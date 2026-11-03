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

func registerReverse(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "reverse-journal-entry",
		Method:      http.MethodPost,
		Path:        "/api/journal-entries/{id}/reverse",
		Summary:     "Reverse a posted journal entry with a new entry that undoes it, now",
		Description: "An entry is reversed at most once: reversing it again is a 409. A reversal that leaves a holding " +
			"negative at some point in time is made all the same, and the response warns about it.",
		Tags:   tags,
		Errors: []int{http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *ReverseJournalEntryInput) (*ReverseJournalEntryOutput, error) {
		result, err := gateway.RunWorkflow[lt.ReverseJournalEntryResult](ctx, c, journalwf.ReverseJournalEntry,
			lt.ReverseJournalEntryInput{JournalEntryID: in.ID})
		if err != nil {
			return nil, err
		}
		return &ReverseJournalEntryOutput{Body: Reversal{
			JournalEntryID: result.JournalEntryID, Warnings: gateway.NewNegativeHoldings(result.Warnings),
		}}, nil
	})
}
