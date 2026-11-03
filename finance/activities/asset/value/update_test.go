package value

import (
	"testing"
	"time"

	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_UpdateAssetValue() {
	now := time.Now()

	newDate := now.Add(24 * time.Hour)
	newValue := decimal.NewFromInt(200)

	asset := &model.Asset{
		Name:   "test asset",
		Ticker: new("TASS"),
		Kind:   model.AssetKindStock,
	}
	_ = s.InsertAsset(s.T(), asset)

	assetValue := &model.AssetValue{
		AssetID:    asset.ID,
		CurrencyID: s.eur.ID,
		Value:      decimal.NewFromInt(100),
		ObservedAt: time.Now(),
	}
	_ = s.InsertAssetValue(s.T(), assetValue)

	tcs := map[string]struct {
		params   avt.UpdateAssetValueParams
		expErr   bool
		expValue decimal.Decimal
		expDate  time.Time
	}{
		"nominal": {
			params: avt.UpdateAssetValueParams{
				ID:         assetValue.ID,
				Value:      &newValue,
				ObservedAt: &newDate,
			},
			expValue: newValue,
			expDate:  newDate,
		},
		"error - no id": {
			params: avt.UpdateAssetValueParams{
				Value: &newValue,
			},
			expErr: true,
		},
		"error - not found": {
			params: avt.UpdateAssetValueParams{
				ID:    999,
				Value: &newValue,
			},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			_, err := s.Env.ExecuteActivity(s.av.UpdateAssetValue, tc.params)

			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// check that the asset value is updated
			var updatedAssetValue model.AssetValue
			err = s.DB.NewSelect().Model(&updatedAssetValue).Where("id = ?", assetValue.ID).Scan(s.T().Context())
			require.NoError(t, err)
			require.True(t, tc.expValue.Equal(updatedAssetValue.Value))
			require.WithinDuration(t, tc.expDate, updatedAssetValue.ObservedAt, time.Second)
		})
	}
}
