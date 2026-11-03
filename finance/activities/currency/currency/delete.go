package currency

import (
	"context"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// DeleteCurrency deletes a currency
func (c *Currency) DeleteCurrency(ctx context.Context, params ct.DeleteCurrencyParams) error {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return err
	}

	params.Code = normaliseCode(params.Code)
	if params.ID == 0 && params.Code == "" {
		return errors.ErrMissingRequiredParameter{Parameters: []string{"ID", "Code"}}
	}

	query := db.NewDelete().Model((*model.Currency)(nil))
	if params.ID != 0 {
		query.Where("id = ?", params.ID)
	} else {
		query.Where("code = ?", params.Code)
	}
	_, err = query.Exec(ctx)
	if conflict := sqllib.StillReferenced(err, "Currency"); conflict != nil {
		return conflict
	}

	return sqllib.IgnoreNotFound(err)
}
