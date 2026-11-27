package presets

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	pt "github.com/logicorg/ivfs/finance/common/types/presettypes"
	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	presetswf "github.com/logicorg/ivfs/finance/workflows/facade/presets"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestApply checks that the body becomes ApplyPreset's input, the tree of
// categories kept, that the result comes back in the same shape, and that
// the workflow's refusals keep their status.
func TestApply(t *testing.T) {
	body := `{
		"currencies": [{"code":"EUR","name":"Euro","symbol":"€","precision":2}],
		"tags": [{"name":"vacation","description":"time off"}],
		"categories": [
			{"name":"Food","kind":"expense","children":[{"name":"Groceries","children":[{"name":"Organic"}]}]},
			{"name":"Salary","kind":"income"}
		]
	}`
	params := pt.ApplyPresetInput{
		Currencies: []ct.InsertCurrencyParams{{Code: "EUR", Name: "Euro", Symbol: "€", Precision: ptr.NewPtr(2)}},
		Tags:       []tt.InsertTagParams{{Name: "vacation", Description: ptr.NewPtr("time off")}},
		Categories: []pt.PresetCategory{
			{Name: "Food", Kind: model.LedgerAccountExpense, Children: []pt.PresetCategory{
				{Name: "Groceries", Children: []pt.PresetCategory{{Name: "Organic"}}},
			}},
			{Name: "Salary", Kind: model.LedgerAccountIncome},
		},
	}
	empty := pt.ApplyPresetInput{Currencies: []ct.InsertCurrencyParams{}, Tags: []tt.InsertTagParams{}, Categories: []pt.PresetCategory{}}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"apply a preset": {
			Method: http.MethodPost, Path: "/api/presets/apply", Body: body,
			Call: &apitest.WorkflowCall{
				Workflow: presetswf.ApplyPreset,
				Params:   params,
				Result: pt.ApplyPresetResult{
					CurrencyIDs: []int64{1},
					TagIDs:      []int64{2},
					Categories: []pt.PresetCategoryResult{
						{CategoryID: 3, Children: []pt.PresetCategoryResult{{CategoryID: 4, Children: []pt.PresetCategoryResult{{CategoryID: 5}}}}},
						{CategoryID: 6},
					},
				},
			},
			WantStatus: http.StatusOK,
			WantBody: `{
				"currencyIds": [1],
				"tagIds": [2],
				"categories": [{"id":3,"children":[{"id":4,"children":[{"id":5}]}]},{"id":6}]
			}`,
		},
		"an empty preset": {
			Method: http.MethodPost, Path: "/api/presets/apply", Body: `{"currencies":[],"tags":[],"categories":[]}`,
			Call: &apitest.WorkflowCall{
				Workflow: presetswf.ApplyPreset,
				Params:   empty,
				Err:      temporal.NewApplicationError("validation error: the preset is empty", "ValidationError"),
			},
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "validation error: the preset is empty"),
		},
		"a currency name that is taken": {
			Method: http.MethodPost, Path: "/api/presets/apply", Body: body,
			Call: &apitest.WorkflowCall{
				Workflow: presetswf.ApplyPreset,
				Params:   params,
				Err:      temporal.NewApplicationError("resource Currency conflict", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource Currency conflict"),
		},
		"a category of an unknown kind": {
			Method: http.MethodPost, Path: "/api/presets/apply",
			Body:       `{"currencies":[],"tags":[],"categories":[{"name":"Cash","kind":"asset"}]}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"a blank tag": {
			Method: http.MethodPost, Path: "/api/presets/apply",
			Body:       `{"currencies":[],"tags":[{"name":""}],"categories":[]}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"no lists": {
			Method: http.MethodPost, Path: "/api/presets/apply", Body: `{}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
