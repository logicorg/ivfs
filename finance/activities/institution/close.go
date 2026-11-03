package institution

import (
	"context"
	"fmt"
	"time"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// Close closes an institution, at ClosedAt or now if it is zero. It is
// idempotent: closing a closed institution succeeds and keeps its ClosedAt. It
// refuses to close an institution that still has open financial accounts;
// those must be closed first, so an open account never sits under a closed
// institution.
func (a *Institution) Close(ctx context.Context, params it.CloseParams) error {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return err
	}

	if err := params.Validate(); err != nil {
		return err
	}

	closedAt := params.ClosedAt
	if closedAt.IsZero() {
		closedAt = time.Now().UTC()
	}

	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		query, findBy := model.FindInstitutionByNameQuery(tx, params.FindBy.Name), any(params.FindBy.Name)
		if params.FindBy.ID > 0 {
			query, findBy = model.FindInstitutionByIDQuery(tx, params.FindBy.ID), params.FindBy.ID
		}
		institution := new(model.Institution)
		if err := query.For("UPDATE").Scan(ctx, institution); err != nil {
			if sqllib.IsNotFound(err) {
				return errors.ErrNotFound{ResourceType: "Institution", ID: findBy}
			}
			return fmt.Errorf("load institution: %w", err)
		}

		if institution.ClosedAt != nil {
			return nil // already closed
		}

		openAccounts, err := tx.NewSelect().Model((*model.FinancialAccount)(nil)).
			Where("fa.institution_id = ?", institution.ID).
			Where("fa.closed_at IS NULL").
			Exists(ctx)
		if err != nil {
			return fmt.Errorf("check open financial accounts: %w", err)
		}
		if openAccounts {
			return errors.ErrResourceConflict{ResourceType: "Institution", Reason: "institution has open financial accounts"}
		}

		institution.ClosedAt = &closedAt
		if _, err := tx.NewUpdate().Model(institution).Column("closed_at", "updated_at").WherePK().Exec(ctx); err != nil {
			return fmt.Errorf("close institution: %w", err)
		}
		return nil
	})
}
