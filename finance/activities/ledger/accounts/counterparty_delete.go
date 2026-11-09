package accounts

import (
	"context"
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// DeleteCounterparty deletes a counterparty, which frees its name (ADR 0016).
// Deleting one that does not exist succeeds, so retries remain idempotent. A
// counterparty that a journal entry still refers to is not deleted: the caller
// gets an ErrResourceConflict and clears the references first.
func (a *Activities) DeleteCounterparty(ctx context.Context, input lt.DeleteCounterpartyInput) error {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	_, err = db.NewDelete().Model((*model.Counterparty)(nil)).Where("id = ?", input.CounterpartyID).Exec(ctx)
	if conflict := sqllib.StillReferenced(err, "Counterparty"); conflict != nil {
		return conflict
	}
	if err != nil {
		return fmt.Errorf("delete counterparty: %w", err)
	}
	return nil
}
