package currency

import (
	"context"
	"database/sql"
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// RetrieveCurrencyByCode retrieves a currency by code, in any casing.
func (c *Currency) RetrieveCurrencyByCode(ctx context.Context, code string) (*model.Currency, error) {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	code = normaliseCode(code)
	currency := &model.Currency{}
	err = model.FindCurrencyByCodeQuery(db, code).Scan(ctx, currency)
	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound{ResourceType: "Currency", ID: code}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve currency by code: %w", err)
	}
	return currency, nil
}

// RetrieveCurrencyByID retrieves a currency by ID
func (c *Currency) RetrieveCurrencyByID(ctx context.Context, id int64) (*model.Currency, error) {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	currency := &model.Currency{}
	err = model.FindCurrencyByIDQuery(db, id).Scan(ctx, currency)
	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound{ResourceType: "Currency", ID: id}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve currency by ID: %w", err)
	}
	return currency, nil
}

// ListCurrencies retrieves all currencies
func (c *Currency) ListCurrencies(ctx context.Context) ([]*model.Currency, error) {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	var currencies []*model.Currency
	err = db.NewSelect().Model(&currencies).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list currencies: %w", err)
	}
	return currencies, nil
}

// CurrencyExists checks if a currency with the given code, in any casing, exists
func (c *Currency) CurrencyExists(ctx context.Context, code string) (bool, error) {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return false, err
	}

	count, err := model.FindCurrencyByCodeQuery(db, normaliseCode(code)).Count(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check if currency exists: %w", err)
	}
	return count > 0, nil
}

// FindCurrencyBy retrieves a currency given the input parameters
func (c *Currency) FindCurrencyBy(ctx context.Context, params ct.FindCurrencyByParams) (*model.Currency, error) {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	params.Code = normaliseCode(params.Code)
	if err := params.Validate(); err != nil {
		return nil, err
	}

	currency := new(model.Currency)

	query := db.NewSelect().Model(currency)

	if params.Name != "" {
		query.Where("name = ?", params.Name)
	}
	if params.Code != "" {
		query.Where("code = ?", params.Code)
	}
	if params.Symbol != "" {
		query.Where("symbol = ?", params.Symbol)
	}

	err = query.Scan(ctx, currency)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound{
				ResourceType: "Currency",
				ID:           fmt.Sprintf("Name: '%s', Code: '%s', Symbol: '%s'", params.Name, params.Code, params.Symbol),
			}
		}
		return nil, fmt.Errorf("failed to find currency: %w", err)
	}

	return currency, nil
}
