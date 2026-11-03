package currency

import (
	"context"
	"database/sql"
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// defaultPrecision is the number of decimals given to a currency inserted
// without one.
const defaultPrecision = 2

// InsertCurrency inserts a new currency into the database with the provided parameters.
func (c *Currency) InsertCurrency(ctx context.Context, param ct.InsertCurrencyParams) (*ct.InsertCurrencyResult, error) {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	// The code is normalised first, so that a code in any casing or spacing
	// passes the upper-case check and is retrieved by its canonical form.
	param.Code = normaliseCode(param.Code)
	if err := param.Validate(); err != nil {
		return nil, err
	}

	currency := &model.Currency{
		Name:   param.Name,
		Code:   param.Code,
		Symbol: param.Symbol,
		// Default precision is applied here rather than as a column default because a
		// zero precision, as JPY has, is a valid value and a default cannot tell it
		// apart from an unset field.
		Precision: defaultPrecision,
	}
	if param.Precision != nil {
		currency.Precision = *param.Precision
	}

	existingCurrency := &model.Currency{}
	err = model.FindCurrencyByCodeQuery(db, param.Code).Scan(ctx, existingCurrency)
	switch err {
	case nil:
		// Currency already exists, return OK
		return &ct.InsertCurrencyResult{CurrencyID: existingCurrency.ID}, nil
	case sql.ErrNoRows:
		// Currency does not exist, proceed with insertion
	default:
		return nil, fmt.Errorf("failed to check for existing currency: %w", err)
	}

	_, err = db.NewInsert().Model(currency).Returning("id").Exec(ctx)
	if conflict := sqllib.AlreadyExists(err, "Currency"); conflict != nil {
		// The code was checked above, so this is a new code with the name of
		// another currency, which is unique too.
		return nil, conflict
	}
	if err != nil {
		return nil, fmt.Errorf("failed to insert currency: %w", err)
	}

	return &ct.InsertCurrencyResult{CurrencyID: currency.ID}, nil
}
