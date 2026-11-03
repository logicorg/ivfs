package asset

import (
	"testing"
	"time"

	"github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_DeleteAsset() {
	assetModel := (*model.Asset)(nil)
	asset := &model.Asset{
		Name:   "test asset",
		Ticker: new("TASS"),
		Kind:   model.AssetKindStock,
	}

	tcs := map[string]struct {
		params               assettypes.DeleteAssetParams
		expErr               bool
		skipAssetExistsCheck bool
	}{
		"nominal by ID": {
			params: assettypes.DeleteAssetParams{ID: 1},
		},
		"nominal by Name": {
			params: assettypes.DeleteAssetParams{Name: "test asset"},
		},
		"nominal by Ticker": {
			params: assettypes.DeleteAssetParams{Ticker: "TASS"},
		},
		"error invalid id": {
			params: assettypes.DeleteAssetParams{ID: 0},
			expErr: true,
		},
		"non-existing id": {
			params: assettypes.DeleteAssetParams{ID: 999},
			// skip as the asset with ID 1 is left untouched in this test case
			skipAssetExistsCheck: true,
		},
		"error no params": {
			params: assettypes.DeleteAssetParams{},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		temporaltesting.Insert(s.T(), s.DB, asset)
		id := asset.ID

		s.T().Run(s.T().Name()+"/"+name, func(t *testing.T) {
			_, err := s.Env.ExecuteActivity(s.aa.DeleteAsset, tc.params)

			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			var exists bool
			var errExists error
			query := s.DB.NewSelect().Model(assetModel).Where("id = ?", id)

			exists, errExists = query.Exists(s.T().Context())
			require.NoError(t, errExists)
			if !tc.skipAssetExistsCheck {
				require.False(t, exists)
			}
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, assetModel, id)
	}
}

func (s *UnitTestSuite) Test_DeleteAsset_Idempotency() {
	asset := &model.Asset{
		Name:   "test asset idempotency",
		Ticker: new("TASI"),
		Kind:   model.AssetKindStock,
	}
	s.InsertAsset(s.T(), asset)

	// delete
	_, err := s.Env.ExecuteActivity(s.aa.DeleteAsset, assettypes.DeleteAssetParams{ID: asset.ID})
	require.NoError(s.T(), err)

	// delete again
	_, err = s.Env.ExecuteActivity(s.aa.DeleteAsset, assettypes.DeleteAssetParams{ID: asset.ID})
	require.NoError(s.T(), err)
}

// An asset that has recorded values cannot be deleted.
func (s *UnitTestSuite) Test_DeleteAsset_InUse() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	asset := &model.Asset{Name: "in use", Ticker: new("USED"), Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)
	temporaltesting.Insert(s.T(), s.DB, &model.AssetValue{AssetID: asset.ID, CurrencyID: eur.ID, Value: decimal.NewFromInt(10), ObservedAt: time.Now()})

	_, err := s.Env.ExecuteActivity(s.aa.DeleteAsset, assettypes.DeleteAssetParams{ID: asset.ID})

	var appErr *temporal.ApplicationError
	require.ErrorAs(s.T(), err, &appErr)
	require.Equal(s.T(), "ErrResourceConflict", appErr.Type())

	exists, err := s.DB.NewSelect().Model((*model.Asset)(nil)).Where("id = ?", asset.ID).Exists(s.T().Context())
	require.NoError(s.T(), err)
	require.True(s.T(), exists)
}
