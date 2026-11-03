package conversion

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// UpdateCurrencyConversion updates a currency conversion
func (c *Conversion) UpdateCurrencyConversion(ctx context.Context, id int64, newRate decimal.Decimal) error {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return err
	}

	if id < 0 {
		return errors.ErrInvalidParameter{Description: "ID must be a positive integer"}
	}
	if newRate.IsNegative() || newRate.IsZero() {
		return errors.ErrInvalidParameter{Description: "Exchange rate must be a positive number"}
	}

	conversion := &model.Conversion{}
	if err := db.NewSelect().Model(conversion).Where("id = ?", id).Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return errors.ErrNotFound{ResourceType: "Currency Conversion", ID: id}
		}
		return fmt.Errorf("failed to retrieve currency conversion: %w", err)
	}

	conversion.Rate = newRate

	if _, err := db.NewUpdate().Model(conversion).WherePK().Exec(ctx); err != nil {
		return fmt.Errorf("failed to update currency conversion: %w", err)
	}
	return nil
}
