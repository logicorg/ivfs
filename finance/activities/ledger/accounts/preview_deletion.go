package accounts

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/deletion"
	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// previewPageSize is how many entries, or movement groups, the preview loads
// at a time.
const previewPageSize = 500

// PreviewAccountDeletion reports what deleting the account would do, and
// changes nothing (ADR 0012). It reads one snapshot and decides each entry and
// movement group as the batches do: which are deleted, which are kept with
// their lines moved to the strays, what lands on the strays, and what would
// refuse the deletion. An institution's financial accounts are previewed as
// one deletion, which ends where deleting them in turn does.
//
// A target that does not exist is ErrNotFound.
func (a *Activities) PreviewAccountDeletion(ctx context.Context, input lt.PreviewAccountDeletionInput) (*lt.AccountDeletionPreview, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	preview := &lt.AccountDeletionPreview{}
	opts := &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}
	err = db.RunInTx(ctx, opts, func(ctx context.Context, tx bun.Tx) error {
		target, conflicts, err := loadPreviewTargetTx(ctx, tx, input)
		if err != nil {
			return err
		}
		tally := &deletionTally{target: target, preview: preview, amounts: map[strayKey]decimal.Decimal{}, units: map[int64]decimal.Decimal{}}
		if err := tally.entriesTx(ctx, tx); err != nil {
			return err
		}
		if err := tally.movementGroupsTx(ctx, tx); err != nil {
			return err
		}
		// The batches refuse a shared or stray target only once it has lines.
		if target.refusal != nil && preview.DeletedEntries+preview.KeptEntries+preview.DeletedMovementGroups+preview.KeptMovementGroups > 0 {
			if conflicts, err = appendConflict(conflicts, target.refusal); err != nil {
				return err
			}
		}
		preview.Conflicts = conflicts
		return tally.strays(ctx, tx)
	})
	if err != nil {
		return nil, err
	}
	return preview, nil
}

// loadPreviewTargetTx loads what the deletion would empty, with the conflicts
// the close before the batches would refuse it with.
func loadPreviewTargetTx(ctx context.Context, tx bun.Tx, input lt.PreviewAccountDeletionInput) (*deletionTarget, []string, error) {
	switch {
	case input.LedgerAccountID != nil:
		target, err := loadDeletionTargetTx(ctx, tx, lt.DeletionTarget{LedgerAccountID: input.LedgerAccountID})
		if err != nil {
			return nil, nil, err
		}
		if target == nil {
			return nil, nil, internalerrors.ErrNotFound{ResourceType: "LedgerAccount", ID: *input.LedgerAccountID}
		}
		conflicts, err := appendConflict(nil, refuseChildAccountsTx(ctx, tx, *input.LedgerAccountID))
		return target, conflicts, err

	case input.FinancialAccountID != nil:
		target, err := loadDeletionTargetTx(ctx, tx, lt.DeletionTarget{FinancialAccountID: input.FinancialAccountID})
		if err != nil {
			return nil, nil, err
		}
		if target == nil {
			return nil, nil, internalerrors.ErrNotFound{ResourceType: "FinancialAccount", ID: *input.FinancialAccountID}
		}
		conflicts, err := appendConflict(nil, refuseForeignChildrenTx(ctx, tx, *input.FinancialAccountID, target.ledgerIDs()))
		return target, conflicts, err
	}

	var institution model.Institution
	err := tx.NewSelect().Model(&institution).Where("i.id = ?", *input.InstitutionID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, internalerrors.ErrNotFound{ResourceType: "Institution", ID: *input.InstitutionID}
	}
	if err != nil {
		return nil, nil, fmt.Errorf("load institution: %w", err)
	}
	var accountIDs []int64
	if err := tx.NewSelect().Model((*model.FinancialAccount)(nil)).Column("fa.id").
		Where("fa.institution_id = ?", institution.ID).OrderExpr("fa.id").
		Scan(ctx, &accountIDs); err != nil {
		return nil, nil, fmt.Errorf("list financial accounts of institution %d: %w", institution.ID, err)
	}

	// Only the stray financial account has a refusal, and it lives under
	// "Untracked", which is refused as a whole.
	target := &deletionTarget{ledgerAccounts: map[int64]*model.LedgerAccount{}, financialAccounts: map[int64]*model.FinancialAccount{}}
	var conflicts []string
	if len(accountIDs) > 0 && lt.IsUntrackedInstitution(institution.Name) {
		conflicts = append(conflicts, fmt.Sprintf("%q is managed by the ledger and cannot be deleted while it holds financial accounts", institution.Name))
	}
	for _, id := range accountIDs {
		account, err := loadDeletionTargetTx(ctx, tx, lt.DeletionTarget{FinancialAccountID: &id})
		if err != nil {
			return nil, nil, err
		}
		for ledgerID, ledgerAccount := range account.ledgerAccounts {
			target.ledgerAccounts[ledgerID] = ledgerAccount
		}
		target.financialAccounts[id] = account.financialAccount
		// The accounts are deleted in turn, so a child in another of the
		// institution's accounts can still refuse one.
		if conflicts, err = appendConflict(conflicts, refuseForeignChildrenTx(ctx, tx, id, account.ledgerIDs())); err != nil {
			return nil, nil, err
		}
	}
	return target, conflicts, nil
}

// appendConflict appends the reason of a conflict, and returns any other error.
func appendConflict(conflicts []string, err error) ([]string, error) {
	var conflict internalerrors.ErrResourceConflict
	if errors.As(err, &conflict) {
		return append(conflicts, conflict.Reason), nil
	}
	return conflicts, err
}

// strayKey is a stray ledger account, which may not exist yet.
type strayKey struct {
	name       string
	kind       model.LedgerAccountKind
	currencyID int64
}

// deletionTally adds up what the deletion would do to each entry and group.
type deletionTally struct {
	target  *deletionTarget
	preview *lt.AccountDeletionPreview
	// amounts is what lands on each stray ledger account, debits minus credits.
	amounts map[strayKey]decimal.Decimal
	// units is what lands on the stray financial account, by asset.
	units map[int64]decimal.Decimal
}

func (d *deletionTally) entriesTx(ctx context.Context, tx bun.Tx) error {
	for after := int64(0); ; {
		ids, err := d.target.entriesTx(ctx, tx, after, previewPageSize)
		if err != nil || len(ids) == 0 {
			return err
		}
		after = ids[len(ids)-1]

		var entries []model.JournalEntry
		if err := tx.NewSelect().Model(&entries).
			Relation("Postings").Relation("Postings.LedgerAccount").
			Relation("PositionMovements").Relation("PositionMovements.FinancialAccount").
			Where("je.id IN (?)", bun.List(ids)).
			Scan(ctx); err != nil {
			return fmt.Errorf("load journal entries: %w", err)
		}
		for i := range entries {
			if err := d.entryTx(ctx, tx, &entries[i]); err != nil {
				return err
			}
		}
	}
}

func (d *deletionTally) entryTx(ctx context.Context, tx bun.Tx, entry *model.JournalEntry) error {
	keep, err := d.target.keepsTx(ctx, tx, entry)
	if err != nil {
		return err
	}
	if !keep {
		d.preview.DeletedEntries++
		return nil
	}
	d.preview.KeptEntries++
	for _, posting := range entry.Postings {
		account, deleted := d.target.ledgerAccounts[posting.LedgerAccountID]
		if !deleted {
			continue
		}
		stray := deletion.StrayLedgerAccount(account.Kind, account.CurrencyID)
		key := strayKey{stray.Name, stray.Kind, stray.CurrencyID}
		amount := posting.Amount
		if posting.Side == model.PostingCredit {
			amount = amount.Neg()
		}
		d.amounts[key] = d.amounts[key].Add(amount)
	}
	for _, movement := range entry.PositionMovements {
		if d.target.isFinancialAccount(movement.FinancialAccountID) {
			d.units[movement.AssetID] = d.units[movement.AssetID].Add(movement.UnitsDelta)
		}
	}
	return nil
}

func (d *deletionTally) movementGroupsTx(ctx context.Context, tx bun.Tx) error {
	if len(d.target.financialAccounts) == 0 {
		return nil
	}
	for offset := 0; ; offset += previewPageSize {
		keys, err := d.target.movementGroupsTx(ctx, tx, offset, previewPageSize)
		if err != nil || len(keys) == 0 {
			return err
		}
		for _, key := range keys {
			group, err := positions.FindByIdempotencyKeyTx(ctx, tx, key)
			if err != nil {
				return err
			}
			if err := d.groupTx(ctx, tx, group); err != nil {
				return err
			}
		}
	}
}

func (d *deletionTally) groupTx(ctx context.Context, tx bun.Tx, group []model.PositionMovement) error {
	accounts, err := financialAccountsTx(ctx, tx, deletion.AccountIDsOf(group))
	if err != nil {
		return err
	}
	keep, err := d.target.keepsGroupTx(ctx, tx, group, accounts)
	if err != nil {
		return err
	}
	if !keep {
		d.preview.DeletedMovementGroups++
		return nil
	}
	d.preview.KeptMovementGroups++
	// A split or adjustment leg on the target is deleted, not moved.
	for _, movement := range group {
		if d.target.isFinancialAccount(movement.FinancialAccountID) && movement.Kind == model.PositionMovementTransfer {
			d.units[movement.AssetID] = d.units[movement.AssetID].Add(movement.UnitsDelta)
		}
	}
	return nil
}

func financialAccountsTx(ctx context.Context, tx bun.Tx, ids []int64) (map[int64]*model.FinancialAccount, error) {
	var accounts []model.FinancialAccount
	if err := tx.NewSelect().Model(&accounts).Where("fa.id IN (?)", bun.List(ids)).Scan(ctx); err != nil {
		return nil, fmt.Errorf("load financial accounts: %w", err)
	}
	byID := make(map[int64]*model.FinancialAccount, len(accounts))
	for i := range accounts {
		byID[accounts[i].ID] = &accounts[i]
	}
	return byID, nil
}

// strays fills the preview's stray amounts and units, leaving out what nets to
// zero, ordered by currency and name, and by asset name.
func (d *deletionTally) strays(ctx context.Context, tx bun.Tx) error {
	var currencyIDs, assetIDs []int64
	for key, amount := range d.amounts {
		if !amount.IsZero() {
			currencyIDs = append(currencyIDs, key.currencyID)
		}
	}
	for assetID, units := range d.units {
		if !units.IsZero() {
			assetIDs = append(assetIDs, assetID)
		}
	}

	if len(currencyIDs) > 0 {
		var currencies []model.Currency
		if err := tx.NewSelect().Model(&currencies).Where("c.id IN (?)", bun.List(currencyIDs)).Scan(ctx); err != nil {
			return fmt.Errorf("load currencies: %w", err)
		}
		codes := map[int64]string{}
		for _, currency := range currencies {
			codes[currency.ID] = currency.Code
		}
		for key, amount := range d.amounts {
			if !amount.IsZero() {
				d.preview.StrayAmounts = append(d.preview.StrayAmounts, lt.StrayAmount{
					Name: key.name, Kind: key.kind, CurrencyCode: codes[key.currencyID], Amount: amount,
				})
			}
		}
		slices.SortFunc(d.preview.StrayAmounts, func(a, b lt.StrayAmount) int {
			return cmp.Or(cmp.Compare(a.CurrencyCode, b.CurrencyCode), cmp.Compare(a.Name, b.Name), cmp.Compare(a.Kind, b.Kind))
		})
	}

	if len(assetIDs) > 0 {
		var assets []model.Asset
		if err := tx.NewSelect().Model(&assets).Where("?TableAlias.id IN (?)", bun.List(assetIDs)).Scan(ctx); err != nil {
			return fmt.Errorf("load assets: %w", err)
		}
		for _, asset := range assets {
			d.preview.StrayUnits = append(d.preview.StrayUnits, lt.StrayUnits{AssetID: asset.ID, AssetName: asset.Name, Units: d.units[asset.ID]})
		}
		slices.SortFunc(d.preview.StrayUnits, func(a, b lt.StrayUnits) int {
			return cmp.Or(cmp.Compare(a.AssetName, b.AssetName), cmp.Compare(a.AssetID, b.AssetID))
		})
	}
	return nil
}
