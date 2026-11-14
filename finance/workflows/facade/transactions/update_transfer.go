package transactions

import (
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	"go.temporal.io/sdk/workflow"
)

// UpdateTransfer makes a journal entry a transfer with the given fields,
// whatever it was before, with the checks of RecordTransfer. The entry keeps
// its id, and loses any position movement it had, so the holdings left
// negative are returned as warnings.
//
// The entry can switch between one currency and two. Between currencies it
// takes the postings and the metadata of an FX conversion without a fee or a
// provider reference, so the rate is worked out again from the new amounts, and
// a fee or a reference the entry had is dropped.
func UpdateTransfer(ctx workflow.Context, input tt.UpdateTransferInput) (*tt.UpdateResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	m, conversion, err := transferOf(ctx, input.Transfer())
	if err != nil {
		return nil, err
	}
	if conversion == nil {
		return replace(ctx, input.JournalEntryID, transferKind, m.entry())
	}
	if err := conversion.Validate(); err != nil {
		return nil, err
	}
	return replace(ctx, input.JournalEntryID, transferKind, conversion.ToJournalEntry())
}
