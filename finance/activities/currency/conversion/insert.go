package conversion

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/conversiontypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/activity"
)

// InsertCurrencyValue inserts a currency value
func (v *Conversion) InsertCurrencyValue(ctx context.Context, params ct.InsertCurrencyValueParams) (*ct.InsertCurrencyValueResult, error) {
	db, err := v.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	logger := activity.GetLogger(ctx)

	if params.FromCurrencyID == 0 || params.ToCurrencyID == 0 || params.ExchangeRate <= 0 {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"FromCurrencyID", "ToCurrencyID", "ExchangeRate"}}
	}

	date := params.EffectiveDate.Truncate(24 * time.Hour).UTC()

	conversion := &model.Conversion{}
	err = db.NewSelect().Model(conversion).Where("from_currency_id = ?", params.FromCurrencyID).Where("to_currency_id = ?", params.ToCurrencyID).Where("observed_at = ?", date).Scan(ctx)

	switch err {
	case sql.ErrNoRows:
		// no conversion found, proceed with insert
		conversion = &model.Conversion{
			FromCurrencyID: params.FromCurrencyID,
			ToCurrencyID:   params.ToCurrencyID,
			Rate:           decimal.NewFromFloat(params.ExchangeRate),
			ObservedAt:     date,
		}

		return v.insert(ctx, conversion)
	case nil:
		logger.Debug("equivalent conversion already exists", "id", conversion.ID)
		return &ct.InsertCurrencyValueResult{ID: conversion.ID}, nil

	default:
		return nil, fmt.Errorf("failed to check for existing currency value: %w", err)
	}
}

func (v *Conversion) insert(ctx context.Context, conversion *model.Conversion) (*ct.InsertCurrencyValueResult, error) {
	db, err := v.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	_, err = db.NewInsert().Model(conversion).Returning("id").Exec(ctx)
	if err != nil {
		if notFound := sqllib.ParentNotFound(err); notFound != nil {
			return nil, notFound
		}
		return nil, fmt.Errorf("failed to insert currency value: %w", err)
	}
	return &ct.InsertCurrencyValueResult{ID: conversion.ID}, nil
}
