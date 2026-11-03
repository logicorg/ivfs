package value

import (
	"time"

	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_DeleteAssetValue() {
	asset := &model.Asset{
		Name:   "test asset",
		Ticker: new("TASS"),
		Kind:   model.AssetKindStock,
	}
	s.InsertAsset(s.T(), asset)

	tcs := map[string]struct {
		params func(assetValue *model.AssetValue) avt.DeleteAssetValueParams
		expErr bool
	}{
		"nominal by ID": {
			params: func(assetValue *model.AssetValue) avt.DeleteAssetValueParams {
				return avt.DeleteAssetValueParams{ID: assetValue.ID}
			},
		},
		"nominal by AssetID": {
			params: func(assetValue *model.AssetValue) avt.DeleteAssetValueParams {
				return avt.DeleteAssetValueParams{AssetID: assetValue.AssetID}
			},
		},
		"error no params": {
			params: func(*model.AssetValue) avt.DeleteAssetValueParams { return avt.DeleteAssetValueParams{} },
			expErr: true,
		},
	}

	for name, tc := range tcs {
		s.Run(name, func() {
			assetValue := &model.AssetValue{
				AssetID:    asset.ID,
				CurrencyID: s.eur.ID,
				Value:      decimal.NewFromInt(100),
				ObservedAt: time.Now(),
			}
			s.InsertAssetValue(s.T(), assetValue)

			_, err := s.Env.ExecuteActivity(s.av.DeleteAssetValue, tc.params(assetValue))
			if tc.expErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)

			exists, err := s.DB.NewSelect().Model((*model.AssetValue)(nil)).Where("id = ?", assetValue.ID).Exists(s.T().Context())
			require.NoError(s.T(), err)
			require.False(s.T(), exists)
		})
	}
}

func (s *UnitTestSuite) Test_DeleteAssetValue_Idempotency() {
	asset := &model.Asset{
		Name:   "test asset idempotency",
		Ticker: new("TASI"),
		Kind:   model.AssetKindStock,
	}
	s.InsertAsset(s.T(), asset)

	assetValue := &model.AssetValue{
		AssetID:    asset.ID,
		CurrencyID: s.eur.ID,
		Value:      decimal.NewFromInt(100),
		ObservedAt: time.Now(),
	}
	s.InsertAssetValue(s.T(), assetValue)

	// delete
	_, err := s.Env.ExecuteActivity(s.av.DeleteAssetValue, avt.DeleteAssetValueParams{ID: assetValue.ID})
	require.NoError(s.T(), err)

	// delete again
	_, err = s.Env.ExecuteActivity(s.av.DeleteAssetValue, avt.DeleteAssetValueParams{ID: assetValue.ID})
	require.NoError(s.T(), err)
}
