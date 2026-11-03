package asset

import (
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

// Test_InsertAsset checks that an asset is stored with its kind, other when
// none is given, that a blank name is refused, and that inserting a name
// again returns the asset already stored, unchanged.
func (s *UnitTestSuite) Test_InsertAsset() {
	tests := map[string]struct {
		existing  *model.Asset
		params    at.InsertAssetParams
		want      model.Asset
		wantError string
	}{
		"an asset with a kind": {
			params: at.InsertAssetParams{Name: "Vanguard FTSE All-World", Kind: model.AssetKindETF, Ticker: ptr.NewPtr("VWCE")},
			want:   model.Asset{Name: "Vanguard FTSE All-World", Kind: model.AssetKindETF, Ticker: ptr.NewPtr("VWCE")},
		},
		"an asset without a kind": {
			params: at.InsertAssetParams{Name: "Gold", Description: ptr.NewPtr("physical")},
			want:   model.Asset{Name: "Gold", Kind: model.AssetKindOther, Description: ptr.NewPtr("physical")},
		},
		"a name already stored": {
			existing: &model.Asset{Name: "Apple Inc.", Kind: model.AssetKindStock, Ticker: ptr.NewPtr("AAPL")},
			params:   at.InsertAssetParams{Name: "Apple Inc.", Kind: model.AssetKindETF},
			want:     model.Asset{Name: "Apple Inc.", Kind: model.AssetKindStock, Ticker: ptr.NewPtr("AAPL")},
		},
		"a blank name": {
			params:    at.InsertAssetParams{Name: " ", Kind: model.AssetKindStock},
			wantError: "missing required parameters: Name",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			if tc.existing != nil {
				temporaltesting.Insert(s.T(), s.DB, tc.existing)
			}

			// A second run is a retry: it must return the same asset and store
			// nothing more.
			var ids []int64
			for range 2 {
				activityResult, err := s.Env.ExecuteActivity(s.aa.InsertAsset, tc.params)
				if tc.wantError != "" {
					require.ErrorContains(s.T(), err, tc.wantError)
					return
				}
				require.NoError(s.T(), err)
				var result at.InsertAssetResult
				require.NoError(s.T(), activityResult.Get(&result))
				ids = append(ids, result.ID)
			}
			require.Equal(s.T(), ids[0], ids[1])
			if tc.existing != nil {
				require.Equal(s.T(), tc.existing.ID, ids[0])
			}

			var stored []model.Asset
			require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("name = ?", tc.params.Name).Scan(s.T().Context()))
			require.Len(s.T(), stored, 1)
			require.Equal(s.T(), tc.want.Name, stored[0].Name)
			require.Equal(s.T(), tc.want.Kind, stored[0].Kind)
			require.Equal(s.T(), tc.want.Ticker, stored[0].Ticker)
			require.Equal(s.T(), tc.want.Description, stored[0].Description)

			temporaltesting.Delete(s.T(), s.DB, (*model.Asset)(nil), ids[0])
		})
	}
}
