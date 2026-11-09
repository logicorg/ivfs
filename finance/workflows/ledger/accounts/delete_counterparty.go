package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// deleteCounterpartyRounds bounds how often DeleteCounterparty clears and
// deletes. A conflict needs an entry naming the counterparty to be recorded in
// the moment between the two steps; needing it twice in a row is already far
// fetched.
const deleteCounterpartyRounds = 2

// DeleteCounterparty deletes a counterparty, as ADR 0016 describes: the journal
// entries that name it lose it, keeping their postings, and then it is
// deleted. The two steps share no transaction, so an entry recorded between
// them makes the delete conflict, and the workflow runs both again. Deleting a
// counterparty that does not exist succeeds.
func DeleteCounterparty(ctx workflow.Context, input lt.DeleteCounterpartyInput) (*lt.DeleteCounterpartyResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	result := &lt.DeleteCounterpartyResult{}
	var err error
	for range deleteCounterpartyRounds {
		var cleared lt.ClearCounterpartyReferencesResult
		if err = workflow.ExecuteActivity(ctx, la.ClearCounterpartyReferences,
			lt.ClearCounterpartyReferencesInput{CounterpartyID: input.CounterpartyID}).Get(ctx, &cleared); err != nil {
			return nil, fmt.Errorf("clear counterparty references: %w", err)
		}
		result.Cleared += cleared.Cleared

		err = workflow.ExecuteActivity(ctx, la.DeleteCounterparty, input).Get(ctx, nil)
		switch {
		case err == nil:
			return result, nil
		case workflows.HasErrorType(err, "ErrResourceConflict"):
			continue
		default:
			return nil, fmt.Errorf("delete counterparty: %w", err)
		}
	}
	return nil, fmt.Errorf("delete counterparty: %w", err)
}
