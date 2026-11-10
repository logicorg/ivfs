package conversion

import (
	"context"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/conversiontypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// RetrieveCurrencyConversionRate retrieves a conversion
func (c *Conversion) RetrieveCurrencyConversionRate(
	ctx context.Context,
	params ct.RetrieveRequest,
) (*ct.RetrieveResponse, error) {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.FromCurrency == 0 {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"FromCurrency"}}
	}
	if params.ToCurrency == 0 {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ToCurrency"}}
	}
	if params.Date.IsZero() {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"Date"}}
	}

	query := model.FindConversionByUniquePropertiesQuery(
		db,
		params.FromCurrency,
		params.ToCurrency,
		params.Date,
	)
	if params.WithRelations {
		query = query.Relation("FromCurrency").Relation("ToCurrency")
	}

	conversion := &model.Conversion{}
	err = query.Scan(ctx, conversion)
	if err != nil {
		if sqllib.IsNotFound(err) {
			return nil, errors.ErrNotFound{
				ResourceType: "Conversion",
				Err:          err,
			}
		}
		return nil, err
	}

	return &ct.RetrieveResponse{Conversions: []*model.Conversion{conversion}}, nil
}

// RetrieveMostRecentCurrencyConversionRatesAtDate retrieves the most recent currency conversion rates
// for the specified FromCurrency and ToCurrency that were created on or before the given date.
//
// It returns up to 'limit' conversion records, ordered by date descending (most recent first).
// If no conversions are found, it returns an ErrNotFound error.
//
// The results are sorted by date in descending order, meaning the most recent conversions
// will appear first in the returned slice.
func (c *Conversion) RetrieveMostRecentCurrencyConversionRatesAtDate(
	ctx context.Context,
	params ct.RetrieveRequest,
	limit int64,
) (*ct.RetrieveResponse, error) {
	db, err := c.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 1
	}

	conversions := make([]*model.Conversion, 0, limit)
	query := db.NewSelect().Model(&conversions).
		Where("from_currency_id = ?", params.FromCurrency).
		Where("to_currency_id = ?", params.ToCurrency).
		Where("DATE(conv.observed_at) <= ?", params.Date.Format(time.DateOnly)).
		Order("conv.observed_at DESC").
		Limit(limit)

	if params.WithRelations {
		query.Relation("FromCurrency").Relation("ToCurrency")
	}

	err = query.Scan(ctx)
	if err != nil {
		if sqllib.IsNotFound(err) {
			return nil, errors.ErrNotFound{
				ResourceType: "Conversion",
				Err:          err,
			}
		}
		return nil, err
	}

	if len(conversions) == 0 {
		return nil, errors.ErrNotFound{
			ResourceType: "Conversion",
		}
	}

	return &ct.RetrieveResponse{Conversions: conversions}, nil
}
