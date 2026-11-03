package conversion

import (
	"testing"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/conversiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_InsertCurrencyValue() {
	fromCurrency := testhelpers.USDollar()
	fromCurrencyID := s.InsertCurrency(s.T(), fromCurrency)

	toCurrency := testhelpers.Euro()
	toCurrencyID := s.InsertCurrency(s.T(), toCurrency)

	tcs := map[string]struct {
		params  ct.InsertCurrencyValueParams
		expErr  bool
		errType string
	}{
		"nominal": {
			params: ct.InsertCurrencyValueParams{
				FromCurrencyID: fromCurrencyID,
				ToCurrencyID:   toCurrencyID,
				ExchangeRate:   1.2,
				EffectiveDate:  time.Now(),
			},
		},
		"error - no from currency": {
			params: ct.InsertCurrencyValueParams{
				ToCurrencyID:  toCurrencyID,
				ExchangeRate:  1.2,
				EffectiveDate: time.Now(),
			},
			expErr:  true,
			errType: "ErrMissingRequiredParameter",
		},
		"error - no to currency": {
			params: ct.InsertCurrencyValueParams{
				FromCurrencyID: fromCurrencyID,
				ExchangeRate:   1.2,
				EffectiveDate:  time.Now(),
			},
			expErr:  true,
			errType: "ErrMissingRequiredParameter",
		},
		"error - no exchange rate": {
			params: ct.InsertCurrencyValueParams{
				FromCurrencyID: fromCurrencyID,
				ToCurrencyID:   toCurrencyID,
				EffectiveDate:  time.Now(),
			},
			expErr:  true,
			errType: "ErrMissingRequiredParameter",
		},
	}

	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			act := &Conversion{}
			res, err := s.Env.ExecuteActivity(act.InsertCurrencyValue, tc.params)

			if tc.expErr {
				require.Error(t, err)
				if tc.errType != "" {
					appErr := new(temporal.ApplicationError)
					require.ErrorAs(t, err, &appErr)
					assert.Equal(t, tc.errType, appErr.Type())
				}
			} else {
				require.NoError(t, err)

				actRes := &ct.InsertCurrencyValueResult{}
				err = res.Get(actRes)
				require.NoError(t, err)
				assert.NotZero(t, actRes.ID, "expected ID to be set")
			}
		})
	}
}

func (s *UnitTestSuite) Test_InsertCurrencyValue_TimeZeroed() {
	now := time.Now()

	dateWithNonZeroTime := time.Date(now.Year(), now.Month(), now.Day(), 06, 10, 46, 63, time.UTC)
	dateWithZeroTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	fromCurrency := testhelpers.USDollar()
	fromCurrencyID := s.InsertCurrency(s.T(), fromCurrency)

	toCurrency := testhelpers.Euro()
	toCurrencyID := s.InsertCurrency(s.T(), toCurrency)

	params := ct.InsertCurrencyValueParams{
		FromCurrencyID: fromCurrencyID,
		ToCurrencyID:   toCurrencyID,
		ExchangeRate:   1.2,
		EffectiveDate:  dateWithNonZeroTime,
	}

	act := &Conversion{}

	// insert
	res, err := s.Env.ExecuteActivity(act.InsertCurrencyValue, params)
	require.NoError(s.T(), err)

	activityResult := &ct.InsertCurrencyValueResult{}
	err = res.Get(activityResult)
	require.NoError(s.T(), err)
	require.NotZero(s.T(), activityResult.ID, "expected ID to be set")

	// retrieve and check that time component is zeroed
	inserted := model.Conversion{
		ID: activityResult.ID,
	}
	err = s.DB.NewSelect().Model(&inserted).WherePK().Scan(s.T().Context())
	require.NoError(s.T(), err, "error retrieving currency value")

	assert.Zero(s.T(), inserted.ObservedAt.Sub(dateWithZeroTime), "expected time component to be zeroed")
}

func (s *UnitTestSuite) Test_InsertCurrencyValue_Idempotency() {
	fromCurrency := testhelpers.USDollar()
	fromCurrencyID := s.InsertCurrency(s.T(), fromCurrency)

	toCurrency := testhelpers.Euro()
	toCurrencyID := s.InsertCurrency(s.T(), toCurrency)

	params := ct.InsertCurrencyValueParams{
		FromCurrencyID: fromCurrencyID,
		ToCurrencyID:   toCurrencyID,
		ExchangeRate:   1.2,
		EffectiveDate:  time.Now(),
	}

	act := &Conversion{}

	// insert
	firstInsertActivityResult, err := s.Env.ExecuteActivity(act.InsertCurrencyValue, params)
	require.NoError(s.T(), err)

	firstInsertCurrencyValueResult := &ct.InsertCurrencyValueResult{}
	err = firstInsertActivityResult.Get(firstInsertCurrencyValueResult)
	require.NoError(s.T(), err)

	// insert again
	secondInsertActivityResult, err := s.Env.ExecuteActivity(act.InsertCurrencyValue, params)
	require.NoError(s.T(), err)

	secondInsertCurrencyValueResult := &ct.InsertCurrencyValueResult{}
	err = secondInsertActivityResult.Get(secondInsertCurrencyValueResult)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), firstInsertCurrencyValueResult.ID, secondInsertCurrencyValueResult.ID, "expected returned IDs to be equal")
}

func (s *UnitTestSuite) Test_InsertCurrencyValue_UnknownCurrency() {
	knownCurrencyID := s.InsertCurrency(s.T(), testhelpers.Euro())

	_, err := s.Env.ExecuteActivity(s.co.InsertCurrencyValue, ct.InsertCurrencyValueParams{
		FromCurrencyID: knownCurrencyID,
		ToCurrencyID:   999999,
		ExchangeRate:   1.2,
		EffectiveDate:  time.Now(),
	})

	appErr := new(temporal.ApplicationError)
	require.ErrorAs(s.T(), err, &appErr)
	assert.Equal(s.T(), "ErrNotFound", appErr.Type())
	require.ErrorContains(s.T(), err, "Currency")
}
