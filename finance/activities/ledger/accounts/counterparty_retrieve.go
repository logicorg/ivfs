package accounts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

func (a *Activities) RetrieveCounterparty(ctx context.Context, params lt.RetrieveCounterpartyParams) (*lt.RetrieveCounterpartyResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	if len(params.Name) > 0 {
		counterparty, err := findCounterpartyByName(ctx, db, params.Name)
		switch {
		case err != nil:
			return nil, err
		case counterparty == nil:
			return nil, internalerrors.ErrNotFound{ResourceType: "Counterparty", ID: params.Name}
		default:
			return &lt.RetrieveCounterpartyResult{Counterparty: counterparty}, nil
		}
	}

	counterparty, err := loadCounterparty(ctx, db, params.ID)
	if err != nil {
		return nil, err
	}
	return &lt.RetrieveCounterpartyResult{Counterparty: counterparty}, nil
}

func (a *Activities) ListCounterparties(ctx context.Context, params lt.ListCounterpartiesParams) (*lt.ListCounterpartiesResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	query := db.NewSelect().Model((*model.Counterparty)(nil)).OrderExpr("cp.name ASC")
	if params.NameContains != "" {
		query = query.Where("cp.name ILIKE ?", "%"+params.NameContains+"%")
	}

	counterparties := make([]model.Counterparty, 0)
	if err := query.Scan(ctx, &counterparties); err != nil {
		return nil, fmt.Errorf("list counterparties: %w", err)
	}
	return &lt.ListCounterpartiesResult{Counterparties: counterparties}, nil
}

// loadCounterparty retrieves a Counterparty by ID or returns
// errors.ErrNotFound. db accepts both *bun.DB and bun.Tx.
// findCounterpartyByName returns the counterparty whose name matches name,
// ignoring case and surrounding space, or nil when there is none. A name is
// unique compared this way (ADR 0005).
func findCounterpartyByName(ctx context.Context, db bun.IDB, name string) (*model.Counterparty, error) {
	counterparty := new(model.Counterparty)
	err := db.NewSelect().Model(counterparty).Where("lower(cp.name) = lower(?)", strings.TrimSpace(name)).Scan(ctx)
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("find counterparty by name: %w", err)
	default:
		return counterparty, nil
	}
}

func loadCounterparty(ctx context.Context, db bun.IDB, id int64) (*model.Counterparty, error) {
	counterparty := new(model.Counterparty)

	err := db.NewSelect().Model(counterparty).Where("id = ?", id).Scan(ctx)

	switch {
	case err == sql.ErrNoRows:
		return nil, internalerrors.ErrNotFound{ResourceType: "Counterparty", ID: id}
	case err != nil:
		return nil, fmt.Errorf("load counterparty: %w", err)
	default:
		return counterparty, nil
	}
}
