package conversion

import (
	"testing"
	"time"

	"github.com/logicorg/ivfs/finance/common/types/conversiontypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_DeleteCurrencyConversionByID() {
	fromCurrID := s.InsertCurrency(s.T(), testhelpers.USDollar())
	toCurrID := s.InsertCurrency(s.T(), testhelpers.Euro())

	conversion := &model.Conversion{
		FromCurrencyID: fromCurrID,
		ToCurrencyID:   toCurrID,
		Rate:           decimal.NewFromFloat(1.2),
		ObservedAt:     time.Now().UTC().Truncate(24 * time.Hour),
	}
	temporaltesting.Insert(s.T(), s.DB, conversion)

	tcs := map[string]struct {
		id     int64
		expErr bool
	}{
		"nominal": {
			id: conversion.ID,
		},
		"error invalid id": {
			id:     0,
			expErr: false, // deleting 0 is a no-op
		},
		"non-existing id": {
			id: 999,
		},
	}

	for name, tc := range tcs {
		s.T().Run(s.T().Name()+"/"+name, func(t *testing.T) {
			err := s.co.DeleteCurrencyConversionByID(s.T().Context(), tc.id)

			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tc.id != 0 {
				// check that the conversion is deleted
				var exists bool
				exists, err = s.DB.NewSelect().Model((*model.Conversion)(nil)).Where("id = ?", tc.id).Exists(s.T().Context())
				require.NoError(t, err)
				require.False(t, exists)
			}
		})
	}
}

func (s *UnitTestSuite) Test_DeleteCurrencyConversionByID_Idempotency() {
	fromCurrID := s.InsertCurrency(s.T(), testhelpers.USDollar())
	toCurrID := s.InsertCurrency(s.T(), testhelpers.Euro())

	conversion := &model.Conversion{
		FromCurrencyID: fromCurrID,
		ToCurrencyID:   toCurrID,
		Rate:           decimal.NewFromFloat(1.2),
		ObservedAt:     time.Now().UTC().Truncate(24 * time.Hour),
	}
	temporaltesting.Insert(s.T(), s.DB, conversion)

	// delete
	err := s.co.DeleteCurrencyConversionByID(s.T().Context(), conversion.ID)
	require.NoError(s.T(), err)

	// delete again
	err = s.co.DeleteCurrencyConversionByID(s.T().Context(), conversion.ID)
	require.NoError(s.T(), err)
}

func (s *UnitTestSuite) Test_DeleteCurrencyConversion() {
	fromCurrID := s.InsertCurrency(s.T(), testhelpers.USDollar())
	toCurrID := s.InsertCurrency(s.T(), testhelpers.Euro())
	createdAt := time.Now().UTC().Truncate(24 * time.Hour)

	conversion := &model.Conversion{
		FromCurrencyID: fromCurrID,
		ToCurrencyID:   toCurrID,
		Rate:           decimal.NewFromFloat(1.2),
		ObservedAt:     createdAt,
	}

	tcs := map[string]struct {
		params conversiontypes.DeleteRequest
		expErr bool
	}{
		"nominal": {
			params: conversiontypes.DeleteRequest{
				FromCurrency: fromCurrID,
				ToCurrency:   toCurrID,
				ObservedAt:   createdAt,
			},
		},
		"by id": {
			params: conversiontypes.DeleteRequest{
				ID: 1,
			},
		},
		"error invalid from currency": {
			params: conversiontypes.DeleteRequest{
				FromCurrency: 0,
				ToCurrency:   toCurrID,
				ObservedAt:   createdAt,
			},
			expErr: false, // deleting with 0 is a no-op
		},
		"error invalid to currency": {
			params: conversiontypes.DeleteRequest{
				FromCurrency: fromCurrID,
				ToCurrency:   0,
				ObservedAt:   createdAt,
			},
			expErr: false, // deleting with 0 is a no-op
		},
		"error invalid created_at": {
			params: conversiontypes.DeleteRequest{
				FromCurrency: fromCurrID,
				ToCurrency:   toCurrID,
				ObservedAt:   time.Time{},
			},
			expErr: false, // deleting with zero time is a no-op
		},
		"non-existing conversion": {
			params: conversiontypes.DeleteRequest{
				FromCurrency: fromCurrID,
				ToCurrency:   toCurrID,
				ObservedAt:   time.Now().Add(24 * time.Hour).UTC().Truncate(24 * time.Hour),
			},
		},
	}

	for name, tc := range tcs {
		temporaltesting.Insert(s.T(), s.DB, conversion)
		id := conversion.ID

		s.T().Run(s.T().Name()+"/"+name, func(t *testing.T) {
			err := s.co.DeleteCurrencyConversion(s.T().Context(), tc.params)

			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if !tc.expErr {
				// check that the conversion is deleted
				var exists bool
				exists, err = s.DB.NewSelect().Model((*model.Conversion)(nil)).
					Where("from_currency_id = ?", tc.params.FromCurrency).
					Where("to_currency_id = ?", tc.params.ToCurrency).
					Where("observed_at = ?", tc.params.ObservedAt).
					Exists(s.T().Context())
				require.NoError(t, err)
				require.False(t, exists)
			}
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Conversion)(nil), id)
	}
}

func (s *UnitTestSuite) Test_DeleteCurrencyConversion_Idempotency() {
	fromCurrID := s.InsertCurrency(s.T(), testhelpers.USDollar())
	toCurrID := s.InsertCurrency(s.T(), testhelpers.Euro())
	createdAt := time.Now().UTC().Truncate(24 * time.Hour)

	conversion := &model.Conversion{
		FromCurrencyID: fromCurrID,
		ToCurrencyID:   toCurrID,
		Rate:           decimal.NewFromFloat(1.2),
		ObservedAt:     createdAt,
	}
	temporaltesting.Insert(s.T(), s.DB, conversion)

	params := conversiontypes.DeleteRequest{
		FromCurrency: fromCurrID,
		ToCurrency:   toCurrID,
		ObservedAt:   createdAt,
	}

	// delete
	err := s.co.DeleteCurrencyConversion(s.T().Context(), params)
	require.NoError(s.T(), err)

	// delete again
	err = s.co.DeleteCurrencyConversion(s.T().Context(), params)
	require.NoError(s.T(), err)
}
