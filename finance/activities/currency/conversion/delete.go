package conversion

import (
	"context"

	"github.com/logicorg/ivfs/finance/common/types/conversiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// DeleteCurrencyConversionByID deletes a conversion by ID
func (c *Conversion) DeleteCurrencyConversionByID(ctx context.Context, id int64) error {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return err
	}

	if id == 0 {
		return nil
	}

	_, err = db.NewDelete().Model((*model.Conversion)(nil)).Where("id = ?", id).Exec(ctx)
	return sqllib.IgnoreNotFound(err)
}

// DeleteCurrencyConversion deletes a currency conversion
func (c *Conversion) DeleteCurrencyConversion(ctx context.Context, params conversiontypes.DeleteRequest) error {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return err
	}

	if params.ID != 0 {
		return c.DeleteCurrencyConversionByID(ctx, params.ID)
	}

	if params.FromCurrency == 0 || params.ToCurrency == 0 || params.ObservedAt.IsZero() {
		return nil
	}

	_, err = model.DeleteConversionByUniquePropertiesQuery(db, params.FromCurrency, params.ToCurrency, params.ObservedAt).Exec(ctx)
	return sqllib.IgnoreNotFound(err)
}
