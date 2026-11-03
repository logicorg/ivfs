package value

import (
	"testing"
	"time"

	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_RetrieveAssetValue() {
	asset := &model.Asset{
		Name:   "test asset",
		Ticker: new("TASS"),
		Kind:   model.AssetKindStock,
	}
	s.InsertAsset(s.T(), asset)

	date0 := time.Now().Add(-48 * time.Hour)
	date1 := time.Now().Add(-24 * time.Hour)
	date2 := time.Now()
	assetValue0 := &model.AssetValue{AssetID: asset.ID, CurrencyID: s.eur.ID, Value: decimal.NewFromInt(50), ObservedAt: date0}
	assetValue1 := &model.AssetValue{AssetID: asset.ID, CurrencyID: s.eur.ID, Value: decimal.NewFromInt(100), ObservedAt: date1}
	assetValue2 := &model.AssetValue{AssetID: asset.ID, CurrencyID: s.eur.ID, Value: decimal.NewFromInt(200), ObservedAt: date2}
	s.InsertAssetValue(s.T(), assetValue0)
	s.InsertAssetValue(s.T(), assetValue1)
	s.InsertAssetValue(s.T(), assetValue2)
	betweenValues := date1.Add(time.Hour)
	beforeValues := date0.Add(-time.Hour)

	tcs := map[string]struct {
		params        avt.RetrieveAssetValueParams
		expErr        bool
		expAssetValue *model.AssetValue
	}{
		"by id": {
			params:        avt.RetrieveAssetValueParams{ID: assetValue1.ID},
			expAssetValue: assetValue1,
		},
		"by asset id": {
			params:        avt.RetrieveAssetValueParams{AssetID: asset.ID},
			expAssetValue: assetValue2, // latest
		},
		"by asset id and date": {
			params:        avt.RetrieveAssetValueParams{AssetID: asset.ID, Date: &date1},
			expAssetValue: assetValue1,
		},
		"by asset id and a date between values, the latest before it": {
			params:        avt.RetrieveAssetValueParams{AssetID: asset.ID, Date: &betweenValues},
			expAssetValue: assetValue1,
		},
		"by asset id and a date before any value": {
			params: avt.RetrieveAssetValueParams{AssetID: asset.ID, Date: &beforeValues},
			expErr: true,
		},
		"not found": {
			params: avt.RetrieveAssetValueParams{ID: 999},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			actRes, err := s.Env.ExecuteActivity(s.av.RetrieveAssetValue, tc.params)
			if tc.expErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			var res avt.RetrieveAssetValueResult
			err = actRes.Get(&res)
			require.NoError(t, err)
			require.Equal(t, tc.expAssetValue.ID, res.AssetValue.ID)
			require.Equal(t, tc.expAssetValue.Value, res.AssetValue.Value)
		})
	}
}

func (s *UnitTestSuite) Test_RetrieveAssetValues() {
	asset := &model.Asset{
		Name:   "test asset",
		Ticker: new("TASS"),
		Kind:   model.AssetKindStock,
	}
	s.InsertAsset(s.T(), asset)

	date1 := time.Now().Add(-48 * time.Hour)
	date2 := time.Now().Add(-24 * time.Hour)
	date3 := time.Now()
	assetValue1 := &model.AssetValue{AssetID: asset.ID, CurrencyID: s.eur.ID, Value: decimal.NewFromInt(100), ObservedAt: date1}
	assetValue2 := &model.AssetValue{AssetID: asset.ID, CurrencyID: s.eur.ID, Value: decimal.NewFromInt(200), ObservedAt: date2}
	assetValue3 := &model.AssetValue{AssetID: asset.ID, CurrencyID: s.eur.ID, Value: decimal.NewFromInt(300), ObservedAt: date3}
	s.InsertAssetValue(s.T(), assetValue1)
	s.InsertAssetValue(s.T(), assetValue2)
	s.InsertAssetValue(s.T(), assetValue3)

	tcs := map[string]struct {
		params        avt.RetrieveAssetValuesParams
		expErr        bool
		expLen        int
		expFirstValue decimal.Decimal
	}{
		"all": {
			params:        avt.RetrieveAssetValuesParams{AssetID: asset.ID},
			expLen:        3,
			expFirstValue: decimal.NewFromInt(100),
		},
		"since": {
			params:        avt.RetrieveAssetValuesParams{AssetID: asset.ID, Since: &date2},
			expLen:        2,
			expFirstValue: decimal.NewFromInt(200),
		},
		"not found": {
			params: avt.RetrieveAssetValuesParams{AssetID: 999},
			expLen: 0,
		},
	}

	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			actRes, err := s.Env.ExecuteActivity(s.av.RetrieveAssetValues, tc.params)
			require.NoError(t, err)

			var res avt.RetrieveAssetValuesResult
			err = actRes.Get(&res)
			require.NoError(t, err)
			require.Len(t, res.AssetValues, tc.expLen)
			if tc.expLen > 0 {
				require.True(t, tc.expFirstValue.Equal(res.AssetValues[0].Value))
			}
		})
	}
}
