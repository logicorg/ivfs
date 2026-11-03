package assets

import (
	"net/http"
	"testing"

	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	assetswf "github.com/logicorg/ivfs/finance/workflows/assets"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestCreate checks that the form is passed to InsertAsset with its ticker and
// description only when given, that the asset's id is returned, that a
// refusal keeps its status, and that a form Huma rejects runs no workflow.
func TestCreate(t *testing.T) {
	insert := func(params at.InsertAssetParams, err error) *apitest.WorkflowCall {
		call := &apitest.WorkflowCall{Workflow: assetswf.InsertAsset, Params: params, Err: err}
		if err == nil {
			call.Result = at.InsertAssetResult{ID: 7}
		}
		return call
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"create an asset": {
			Method: http.MethodPost, Path: "/api/assets", Body: `{"name":"Gold","kind":"commodity"}`,
			Call:       insert(at.InsertAssetParams{Name: "Gold", Kind: model.AssetKindCommodity}, nil),
			WantStatus: http.StatusOK, WantBody: `{"id":7}`,
		},
		"with a ticker and a description": {
			Method: http.MethodPost, Path: "/api/assets",
			Body: `{"name":"Vanguard FTSE All-World","kind":"etf","ticker":"VWCE","description":"accumulating"}`,
			Call: insert(at.InsertAssetParams{
				Name: "Vanguard FTSE All-World", Kind: model.AssetKindETF,
				Ticker: ptr.NewPtr("VWCE"), Description: ptr.NewPtr("accumulating"),
			}, nil),
			WantStatus: http.StatusOK, WantBody: `{"id":7}`,
		},
		"a blank name": {
			Method: http.MethodPost, Path: "/api/assets", Body: `{"name":" ","kind":"etf"}`,
			Call: insert(at.InsertAssetParams{Name: " ", Kind: model.AssetKindETF},
				temporal.NewApplicationError("missing required parameters: Name", "ErrMissingRequiredParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "missing required parameters: Name"),
		},
		"no name": {
			Method: http.MethodPost, Path: "/api/assets", Body: `{"kind":"etf"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"a kind that is not one": {
			Method: http.MethodPost, Path: "/api/assets", Body: `{"name":"Gold","kind":"metal"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
