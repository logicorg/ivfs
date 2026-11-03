package value

import (
	"testing"
	"time"

	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_InsertAssetValue() {
	asset := &model.Asset{
		Name:   "test asset",
		Ticker: new("TASS"),
		Kind:   model.AssetKindStock,
	}
	s.InsertAsset(s.T(), asset)

	now := time.Now()

	tcs := map[string]struct {
		params avt.InsertAssetValueParams
		expErr bool
	}{
		"nominal": {
			params: avt.InsertAssetValueParams{
				AssetID:    asset.ID,
				CurrencyID: s.eur.ID,
				Date:       now,
				Value:      decimal.NewFromInt(100),
			},
		},
		"error - no asset id": {
			params: avt.InsertAssetValueParams{
				Value: decimal.NewFromInt(100),

				Date: now,
			},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			actRes, err := s.Env.ExecuteActivity(s.av.InsertAssetValue, tc.params)

			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			var res avt.InsertAssetValueResult
			err = actRes.Get(&res)
			require.NoError(t, err)
			require.NotZero(t, res.ID)
		})
	}
}

func (s *UnitTestSuite) Test_InsertAssetValue_Idempotency() {
	asset := &model.Asset{
		Name:   "test asset idempotency",
		Ticker: new("TASI"),
		Kind:   model.AssetKindStock,
	}
	s.InsertAsset(s.T(), asset)

	now := time.Now()

	insertParams := avt.InsertAssetValueParams{
		AssetID:    asset.ID,
		CurrencyID: s.eur.ID,
		Value:      decimal.NewFromInt(100),
		Date:       now,
	}

	// insert
	res, err := s.Env.ExecuteActivity(s.av.InsertAssetValue, insertParams)
	require.NoError(s.T(), err)

	insertResult := &avt.InsertAssetValueResult{}
	require.NoError(s.T(), res.Get(insertResult))
	require.NotZero(s.T(), insertResult.ID)

	// insert again
	res, err = s.Env.ExecuteActivity(s.av.InsertAssetValue, insertParams)
	require.NoError(s.T(), err)

	secondInsertResult := &avt.InsertAssetValueResult{}
	require.NoError(s.T(), res.Get(secondInsertResult))
	require.Equal(s.T(), insertResult.ID, secondInsertResult.ID)
}

func (s *UnitTestSuite) Test_InsertAssetValue_MissingParents() {
	asset := &model.Asset{Name: "known asset", Ticker: new("KNWN"), Kind: model.AssetKindStock}
	s.InsertAsset(s.T(), asset)

	tests := map[string]struct {
		params       avt.InsertAssetValueParams
		expectedType string
	}{
		"unknown asset": {
			params:       avt.InsertAssetValueParams{AssetID: 999999, CurrencyID: s.eur.ID, Date: time.Now(), Value: decimal.NewFromInt(10)},
			expectedType: "Asset",
		},
		"unknown currency": {
			params:       avt.InsertAssetValueParams{AssetID: asset.ID, CurrencyID: 999999, Date: time.Now(), Value: decimal.NewFromInt(10)},
			expectedType: "Currency",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.av.InsertAssetValue, tc.params)

			var appErr *temporal.ApplicationError
			require.ErrorAs(s.T(), err, &appErr)
			require.Equal(s.T(), "ErrNotFound", appErr.Type())
			require.ErrorContains(s.T(), err, tc.expectedType)
		})
	}
}
