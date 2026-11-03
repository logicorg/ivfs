package asset

import (
	"testing"

	"github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_UpdateAsset() {
	asset := &model.Asset{
		Name:   "test asset",
		Ticker: new("TASS"),
		Kind:   model.AssetKindStock,
	}

	newName := "new name"

	tcs := map[string]struct {
		params     assettypes.UpdateAssetParams
		expErr     bool
		validation func(t *testing.T, updated *model.Asset)
	}{
		"nominal": {
			params: assettypes.UpdateAssetParams{
				FindBy: assettypes.RetrieveAssetParams{ID: 1},
				Name:   ptr.NewPtr(newName),
			},
			validation: func(t *testing.T, updated *model.Asset) {
				assert.Equal(t, newName, updated.Name)
			},
		},
		"error - not found": {
			params: assettypes.UpdateAssetParams{
				FindBy: assettypes.RetrieveAssetParams{ID: 999},
				Name:   ptr.NewPtr(newName),
			},
			expErr: true,
		},
		"error - no findby": {
			params: assettypes.UpdateAssetParams{
				Name: ptr.NewPtr(newName),
			},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		temporaltesting.Insert(s.T(), s.DB, asset)

		s.T().Run(name, func(t *testing.T) {
			_, err := s.Env.ExecuteActivity(s.aa.UpdateAsset, tc.params)

			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// check that the asset is updated
			var updatedAsset model.Asset
			err = s.DB.NewSelect().Model(&updatedAsset).Where("id = ?", asset.ID).Scan(s.T().Context())
			require.NoError(t, err)

			if tc.validation != nil {
				tc.validation(t, &updatedAsset)
			}
		})

		temporaltesting.Delete(s.T(), s.DB, (*model.Asset)(nil), 1)
	}
}
