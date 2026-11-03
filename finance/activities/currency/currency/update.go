package currency

import (
	"context"
	"database/sql"
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// UpdateCurrency updates an existing currency in the database.
func (c *Currency) UpdateCurrency(ctx context.Context, params ct.UpdateCurrencyParams) error {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return err
	}

	if params.ID == 0 {
		return errors.ErrMissingRequiredParameter{Parameters: []string{"ID"}}
	}

	currency := &model.Currency{}
	if err := model.FindCurrencyByIDQuery(db, params.ID).Scan(ctx, currency); err != nil {
		if err == sql.ErrNoRows {
			return errors.ErrNotFound{ResourceType: "Currency", ID: params.ID}
		}
		return fmt.Errorf("failed to retrieve currency: %w", err)
	}

	if params.Name != nil {
		currency.Name = *params.Name
	}
	if params.Code != nil {
		currency.Code = normaliseCode(*params.Code)
	}
	if params.Symbol != nil {
		currency.Symbol = *params.Symbol
	}

	_, err = db.NewUpdate().Model(currency).WherePK().Exec(ctx)
	if conflict := sqllib.AlreadyExists(err, "Currency"); conflict != nil {
		return conflict
	}
	if err != nil {
		return fmt.Errorf("failed to update currency: %w", err)
	}

	return nil
}
