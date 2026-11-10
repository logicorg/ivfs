package accounts

import (
	"context"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/deletion"
	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// HandleMovementGroupsForDeletion handles the next batch of entry-less movement
// groups in the target financial account, up to input.Limit in the order they
// were written, in one transaction (ADR 0012). A group that also moves units on
// a surviving financial account other than the stray one is kept: its transfer
// legs in the target move to the stray financial account, and its split and
// adjustment legs there are deleted alone. Any other group is deleted, as
// DeletePositionMovement does. Either way the group no longer touches the target, so a batch that handles nothing
// means none are left, and a batch run again moves nothing twice.
//
// A ledger account has no movements, and neither has a target that does not
// exist. It refuses the stray financial account with ErrResourceConflict when
// it has groups.
func (a *Activities) HandleMovementGroupsForDeletion(ctx context.Context, input lt.DeletionBatchInput) (*lt.DeletionBatchResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	result := &lt.DeletionBatchResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		target, err := loadDeletionTargetTx(ctx, tx, input.Target)
		if err != nil || target == nil || target.financialAccount == nil {
			return err
		}
		keys, err := target.movementGroupsTx(ctx, tx, 0, input.Limit)
		if err != nil || len(keys) == 0 {
			return err
		}
		if target.refusal != nil {
			return target.refusal
		}

		groups := make([][]model.PositionMovement, 0, len(keys))
		accountIDs := []int64{target.financialAccount.ID}
		for _, key := range keys {
			group, err := positions.FindByIdempotencyKeyTx(ctx, tx, key)
			if err != nil {
				return err
			}
			groups = append(groups, group)
			accountIDs = append(accountIDs, deletion.AccountIDsOf(group)...)
		}
		locked, err := positions.LockFinancialAccountsTx(ctx, tx, accountIDs)
		if err != nil {
			return err
		}

		var kept [][]model.PositionMovement
		for _, group := range groups {
			keep, err := target.keepsGroupTx(ctx, tx, group, locked)
			if err != nil {
				return err
			}
			if keep {
				kept = append(kept, group)
				continue
			}
			if err := deletion.DeleteGroupTx(ctx, tx, *group[0].IdempotencyKey); err != nil {
				return err
			}
		}
		if err := target.moveLegsTx(ctx, tx, kept); err != nil {
			return err
		}
		result.Handled = len(keys)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// movementGroupsTx returns the keys of up to limit entry-less groups, past the
// first offset, that move units in the target's financial accounts, in the
// order of their first movement.
func (t *deletionTarget) movementGroupsTx(ctx context.Context, tx bun.Tx, offset, limit int) ([]string, error) {
	var keys []string
	err := tx.NewSelect().Model((*model.PositionMovement)(nil)).Column("pm.idempotency_key").
		Where("pm.journal_entry_id IS NULL").Where("pm.financial_account_id IN (?)", bun.List(t.financialIDs())).
		Group("pm.idempotency_key").OrderExpr("min(pm.id)").Offset(int64(offset)).Limit(int64(limit)).
		Scan(ctx, &keys)
	if err != nil {
		return nil, fmt.Errorf("list position movement groups in the deleted account: %w", err)
	}
	return keys, nil
}

// keepsGroupTx reports whether the group moves units on a financial account
// that survives, the stray one aside, whose holdings are not tracked. The
// accounts are the locked ones, which include every account of the group.
func (t *deletionTarget) keepsGroupTx(ctx context.Context, tx bun.Tx, group []model.PositionMovement, accounts map[int64]*model.FinancialAccount) (bool, error) {
	for _, movement := range group {
		if t.isFinancialAccount(movement.FinancialAccountID) {
			continue
		}
		stray, err := isStrayFinancialAccountTx(ctx, tx, accounts[movement.FinancialAccountID])
		if err != nil {
			return false, err
		}
		if !stray {
			return true, nil
		}
	}
	return false, nil
}

// moveLegsTx empties the target of the kept groups' legs: each transfer leg
// moves to the stray financial account, and each split or adjustment leg is
// deleted, since it is a fact about the target alone and its group stays valid
// without it.
func (t *deletionTarget) moveLegsTx(ctx context.Context, tx bun.Tx, kept [][]model.PositionMovement) error {
	var moving, dropped []int64
	for _, group := range kept {
		for _, movement := range group {
			if !t.isFinancialAccount(movement.FinancialAccountID) {
				continue
			}
			if movement.Kind == model.PositionMovementTransfer {
				moving = append(moving, movement.ID)
			} else {
				dropped = append(dropped, movement.ID)
			}
		}
	}
	if len(moving) == 0 && len(dropped) == 0 {
		return nil
	}

	if _, err := tx.ExecContext(ctx, `SELECT set_config('ivfs.deleting_financial_account', ?, true)`, fmt.Sprint(t.financialAccount.ID)); err != nil {
		return fmt.Errorf("declare financial account deletion: %w", err)
	}
	if len(dropped) > 0 {
		if _, err := tx.NewDelete().Model((*model.PositionMovement)(nil)).Where("id IN (?)", bun.List(dropped)).Exec(ctx); err != nil {
			return fmt.Errorf("delete position movements: %w", err)
		}
	}
	if len(moving) == 0 {
		return nil
	}
	strayID, err := deletion.StrayFinancialAccountTx(ctx, tx)
	if err != nil {
		return err
	}
	if _, err := tx.NewUpdate().Model((*model.PositionMovement)(nil)).
		Set("financial_account_id = ?", strayID).
		Where("pm.id IN (?)", bun.List(moving)).
		Exec(ctx); err != nil {
		return fmt.Errorf("move position movements: %w", err)
	}
	return nil
}
