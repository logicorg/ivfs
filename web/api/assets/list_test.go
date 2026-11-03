package assets

import (
	"net/http"
	"testing"

	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	assetswf "github.com/logicorg/ivfs/finance/workflows/assets"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestList checks that the filters are passed to ListAssets, that the assets
// are listed as the API's own type with their missing fields omitted, and
// that none is an empty list rather than null.
func TestList(t *testing.T) {
	list := func(params at.ListAssetsParams, assets ...*model.Asset) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{Workflow: assetswf.ListAssets, Params: params, Result: at.ListAssetsResult{Assets: assets}}
	}
	vwce := &model.Asset{
		ID: 7, Name: "Vanguard FTSE All-World", Kind: model.AssetKindETF, Ticker: ptr.NewPtr("VWCE"),
		ISIN: ptr.NewPtr("IE00BK5BQT80"), Exchange: ptr.NewPtr("XETRA"), Description: ptr.NewPtr("accumulating"),
	}
	gold := &model.Asset{ID: 8, Name: "Gold", Kind: model.AssetKindCommodity}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"list the assets": {
			Method: http.MethodGet, Path: "/api/assets",
			Call:       list(at.ListAssetsParams{}, gold, vwce),
			WantStatus: http.StatusOK,
			WantBody: `{"assets":[{"id":8,"name":"Gold","kind":"commodity"},{"id":7,"name":"Vanguard FTSE All-World",` +
				`"kind":"etf","ticker":"VWCE","isin":"IE00BK5BQT80","exchange":"XETRA","description":"accumulating"}]}`,
		},
		"filter them": {
			Method: http.MethodGet, Path: "/api/assets?nameContains=world&tickerContains=vw",
			Call:       list(at.ListAssetsParams{NameContains: "world", TickerContains: "vw"}, vwce),
			WantStatus: http.StatusOK,
			WantBody: `{"assets":[{"id":7,"name":"Vanguard FTSE All-World","kind":"etf","ticker":"VWCE",` +
				`"isin":"IE00BK5BQT80","exchange":"XETRA","description":"accumulating"}]}`,
		},
		"list no assets": {
			Method: http.MethodGet, Path: "/api/assets",
			Call:       list(at.ListAssetsParams{}),
			WantStatus: http.StatusOK, WantBody: `{"assets":[]}`,
		},
	})
}
