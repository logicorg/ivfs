package conversion

import (
	"testing"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/conversiontypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_RetrieveCurrencyConversionRate() {
	fromCurrency := testhelpers.USDollar()
	fromCurrencyID := s.InsertCurrency(s.T(), fromCurrency)

	toCurrency := testhelpers.Euro()
	toCurrencyID := s.InsertCurrency(s.T(), toCurrency)

	conversion := &model.Conversion{
		FromCurrencyID: fromCurrencyID,
		ToCurrencyID:   toCurrencyID,
		Rate:           decimal.NewFromFloat(1.2),
		ObservedAt:     time.Now(),
	}
	temporaltesting.Insert(s.T(), s.DB, conversion)

	tcs := map[string]struct {
		params        ct.RetrieveRequest
		expErr        bool
		errType       string
		expConversion *model.Conversion
	}{
		"nominal": {
			params: ct.RetrieveRequest{
				FromCurrency: fromCurrencyID,
				ToCurrency:   toCurrencyID,
				Date:         conversion.ObservedAt,
			},
			expConversion: conversion,
		},
		"with relations": {
			params: ct.RetrieveRequest{
				FromCurrency:  fromCurrencyID,
				ToCurrency:    toCurrencyID,
				Date:          conversion.ObservedAt,
				WithRelations: true,
			},
			expConversion: conversion,
		},
		"not found": {
			params: ct.RetrieveRequest{
				FromCurrency: fromCurrencyID,
				ToCurrency:   toCurrencyID,
				Date:         time.Now().Add(24 * time.Hour),
			},
			expErr:  true,
			errType: "ErrNotFound",
		},
		"missing from currency": {
			params: ct.RetrieveRequest{
				ToCurrency: toCurrencyID,
				Date:       conversion.ObservedAt,
			},
			expErr:  true,
			errType: "ErrMissingRequiredParameter",
		},
		"missing to currency": {
			params: ct.RetrieveRequest{
				FromCurrency: fromCurrencyID,
				Date:         conversion.ObservedAt,
			},
			expErr:  true,
			errType: "ErrMissingRequiredParameter",
		},
		"missing date": {
			params: ct.RetrieveRequest{
				FromCurrency: fromCurrencyID,
				ToCurrency:   toCurrencyID,
			},
			expErr:  true,
			errType: "ErrMissingRequiredParameter",
		},
	}

	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			act := &Conversion{Pools: tenancy.Static(s.DB)}
			res, err := s.Env.ExecuteActivity(act.RetrieveCurrencyConversionRate, tc.params)

			if tc.expErr {
				require.Error(t, err)
				if tc.errType != "" {
					appErr := new(temporal.ApplicationError)
					require.ErrorAs(t, err, &appErr)
					assert.Equal(t, tc.errType, appErr.Type())
				}
			} else {
				require.NoError(t, err)
				var actRes ct.RetrieveResponse
				err = res.Get(&actRes)
				require.NoError(t, err)
				assert.Equal(t, tc.expConversion.ID, actRes.Conversions[0].ID)
				if tc.params.WithRelations {
					assert.NotNil(t, actRes.Conversions[0].FromCurrency)
					assert.NotNil(t, actRes.Conversions[0].ToCurrency)
				}
			}
		})
	}
}

func (s *UnitTestSuite) Test_RetrieveMostRecentCurrencyConversionRatesAtDate() {
	fromCurrency := testhelpers.USDollar()
	fromCurrencyID := s.InsertCurrency(s.T(), fromCurrency)

	toCurrency := testhelpers.Euro()
	toCurrencyID := s.InsertCurrency(s.T(), toCurrency)

	date1 := time.Now().Add(-48 * time.Hour)
	date2 := time.Now().Add(-24 * time.Hour)
	date3 := time.Now()

	conversion1 := &model.Conversion{FromCurrencyID: fromCurrencyID, ToCurrencyID: toCurrencyID, Rate: decimal.NewFromFloat(1.1), ObservedAt: date1}
	conversion2 := &model.Conversion{FromCurrencyID: fromCurrencyID, ToCurrencyID: toCurrencyID, Rate: decimal.NewFromFloat(1.2), ObservedAt: date2}
	conversion3 := &model.Conversion{FromCurrencyID: fromCurrencyID, ToCurrencyID: toCurrencyID, Rate: decimal.NewFromFloat(1.3), ObservedAt: date3}
	temporaltesting.Insert(s.T(), s.DB, conversion1)
	temporaltesting.Insert(s.T(), s.DB, conversion2)
	temporaltesting.Insert(s.T(), s.DB, conversion3)

	tcs := map[string]struct {
		params        ct.RetrieveRequest
		limit         int64
		expErr        bool
		errType       string
		expLen        int
		expFirstID    int64
		withRelations bool
	}{
		"most recent": {
			params:     ct.RetrieveRequest{FromCurrency: fromCurrencyID, ToCurrency: toCurrencyID, Date: time.Now()},
			limit:      1,
			expLen:     1,
			expFirstID: conversion3.ID,
		},
		"multiple": {
			params:     ct.RetrieveRequest{FromCurrency: fromCurrencyID, ToCurrency: toCurrencyID, Date: time.Now()},
			limit:      2,
			expLen:     2,
			expFirstID: conversion3.ID,
		},
		"not found": {
			params:  ct.RetrieveRequest{FromCurrency: 999, ToCurrency: 998, Date: time.Now()},
			limit:   1,
			expErr:  true,
			errType: "ErrNotFound",
		},
		"with relations": {
			params:        ct.RetrieveRequest{FromCurrency: fromCurrencyID, ToCurrency: toCurrencyID, Date: time.Now(), WithRelations: true},
			limit:         1,
			expLen:        1,
			expFirstID:    conversion3.ID,
			withRelations: true,
		},
	}

	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			act := &Conversion{Pools: tenancy.Static(s.DB)}
			res, err := s.Env.ExecuteActivity(act.RetrieveMostRecentCurrencyConversionRatesAtDate, tc.params, tc.limit)

			if tc.expErr {
				require.Error(t, err)
				if tc.errType != "" {
					appErr := new(temporal.ApplicationError)
					require.ErrorAs(t, err, &appErr)
					assert.Equal(t, tc.errType, appErr.Type())
				}
			} else {
				require.NoError(t, err)
				var actRes *ct.RetrieveResponse
				err = res.Get(&actRes)
				require.NoError(t, err)
				assert.Len(t, actRes.Conversions, tc.expLen)
				if len(actRes.Conversions) > 0 {
					assert.Equal(t, tc.expFirstID, actRes.Conversions[0].ID)
					if tc.withRelations {
						assert.NotNil(t, actRes.Conversions[0].FromCurrency)
						assert.NotNil(t, actRes.Conversions[0].ToCurrency)
					}
				}
			}
		})
	}
}
