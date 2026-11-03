package conversion

import (
	"testing"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_UpdateCurrencyConversion() {
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

	newRate := decimal.RequireFromString("1.3")

	tcs := map[string]struct {
		id      int64
		newRate decimal.Decimal
		expErr  bool
		errType string
	}{
		"nominal": {
			id:      1,
			newRate: newRate,
		},
		"not found": {
			id:      999,
			newRate: newRate,
			expErr:  true,
			errType: "ErrNotFound",
		},
		"invalid id": {
			id:      -1,
			newRate: newRate,
			expErr:  true,
			errType: "ErrInvalidParameter",
		},
		"invalid rate": {
			id:      1,
			newRate: decimal.Decimal{},
			expErr:  true,
			errType: "ErrInvalidParameter",
		},
	}

	for name, tc := range tcs {
		temporaltesting.Insert(s.T(), s.DB, conversion)
		id := conversion.ID

		s.T().Run(name, func(t *testing.T) {
			act := &Conversion{Pools: tenancy.Static(s.DB)}
			_, err := s.Env.ExecuteActivity(act.UpdateCurrencyConversion, tc.id, tc.newRate)

			if tc.expErr {
				require.Error(t, err)
				if tc.errType != "" {
					appErr := new(temporal.ApplicationError)
					require.ErrorAs(t, err, &appErr)
					assert.Equal(t, tc.errType, appErr.Type())
				}
			} else {
				require.NoError(t, err)
				// verify update
				updatedConversion := &model.Conversion{ID: conversion.ID}
				err := s.DB.NewSelect().Model(updatedConversion).WherePK().Scan(s.T().Context())
				require.NoError(t, err)
				assert.True(t, tc.newRate.Equal(updatedConversion.Rate))
			}
		})

		temporaltesting.Delete(s.T(), s.DB, (*model.Conversion)(nil), id)
	}
}
