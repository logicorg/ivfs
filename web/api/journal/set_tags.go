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

func registerSetTags(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "set-journal-entry-tags",
		Method:      http.MethodPut,
		Path:        "/api/journal-entries/{id}/tags",
		Summary:     "Replace the tags of a journal entry",
		Description: "The entry gets exactly the tags given, and an empty list clears them. Its postings are left as " +
			"they are. An unknown entry or tag changes nothing.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusNotFound},
	}, func(ctx context.Context, in *SetJournalEntryTagsInput) (*SetJournalEntryTagsOutput, error) {
		result, err := gateway.RunWorkflow[lt.SetJournalEntryTagsResult](ctx, c, journalwf.SetJournalEntryTags,
			lt.SetJournalEntryTagsInput{JournalEntryID: in.ID, TagIDs: in.Body.TagIDs})
		if err != nil {
			return nil, err
		}
		// An entry without tags is an empty list, not null, so a client need not
		// tell the two apart.
		tagIDs := result.TagIDs
		if tagIDs == nil {
			tagIDs = []int64{}
		}
		return &SetJournalEntryTagsOutput{Body: EntryTagIDs{TagIDs: tagIDs}}, nil
	})
}
