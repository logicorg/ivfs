package asset

import (
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

// Test_ListAssets covers filtering by a case-insensitive substring of the
// asset's name and/or ticker, and that the two filters are ANDed together.
func (s *UnitTestSuite) Test_ListAssets() {
	apple := &model.Asset{Name: "Apple Inc.", Ticker: ptr.NewPtr("AAPL"), Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, apple)
	pineapple := &model.Asset{Name: "Pineapple Fund", Ticker: ptr.NewPtr("PNAP"), Kind: model.AssetKindFund}
	temporaltesting.Insert(s.T(), s.DB, pineapple)
	msft := &model.Asset{Name: "Microsoft Corp.", Ticker: ptr.NewPtr("MSFT"), Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, msft)

	tests := map[string]struct {
		params at.ListAssetsParams
		expIDs []int64
	}{
		"no filter returns everything": {
			params: at.ListAssetsParams{},
			expIDs: []int64{apple.ID, pineapple.ID, msft.ID},
		},
		"filter by name substring, case-insensitive": {
			params: at.ListAssetsParams{NameContains: "apple"},
			expIDs: []int64{apple.ID, pineapple.ID},
		},
		"filter by ticker substring, case-insensitive": {
			params: at.ListAssetsParams{TickerContains: "aapl"},
			expIDs: []int64{apple.ID},
		},
		"name and ticker filters are ANDed": {
			params: at.ListAssetsParams{NameContains: "apple", TickerContains: "pnap"},
			expIDs: []int64{pineapple.ID},
		},
		"no match returns empty": {
			params: at.ListAssetsParams{NameContains: "nonexistent"},
			expIDs: []int64{},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.ListAssets, tc.params)
			require.NoError(s.T(), err)

			var result at.ListAssetsResult
			require.NoError(s.T(), activityResult.Get(&result))
			gotIDs := make([]int64, 0, len(result.Assets))
			for _, a := range result.Assets {
				gotIDs = append(gotIDs, a.ID)
			}
			require.ElementsMatch(s.T(), tc.expIDs, gotIDs)
		})
	}
}
