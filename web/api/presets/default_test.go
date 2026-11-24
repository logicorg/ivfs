package presets

import (
	"net/http"
	"testing"

	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestDefault checks that the default preset is served without a workflow,
// precisions spelled out and children without a kind.
func TestDefault(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"the default preset": {
			Method: http.MethodGet, Path: "/api/presets/default",
			WantStatus: http.StatusOK,
			WantBody: `{
				"currencies": [
					{"code":"EUR","name":"Euro","symbol":"€","precision":2},
					{"code":"USD","name":"US Dollar","symbol":"$","precision":2},
					{"code":"GBP","name":"British Pound","symbol":"£","precision":2},
					{"code":"CHF","name":"Swiss Franc","symbol":"CHF","precision":2},
					{"code":"JPY","name":"Japanese Yen","symbol":"¥","precision":0}
				],
				"tags": [{"name":"vacation"},{"name":"reimbursable"},{"name":"tax-deductible"},{"name":"recurring"}],
				"categories": [
					{"name":"Housing","kind":"expense","children":[{"name":"Rent or mortgage"},{"name":"Utilities"},{"name":"Maintenance"}]},
					{"name":"Food","kind":"expense","children":[{"name":"Groceries"},{"name":"Eating out"}]},
					{"name":"Transport","kind":"expense","children":[{"name":"Public transport"},{"name":"Fuel"},{"name":"Vehicle"}]},
					{"name":"Leisure","kind":"expense","children":[{"name":"Subscriptions"},{"name":"Travel"},{"name":"Hobbies"}]},
					{"name":"Health","kind":"expense"},
					{"name":"Shopping","kind":"expense"},
					{"name":"Taxes","kind":"expense"},
					{"name":"Fees","kind":"expense"},
					{"name":"Salary","kind":"income"},
					{"name":"Interest","kind":"income"},
					{"name":"Dividends","kind":"income"},
					{"name":"Gifts","kind":"income"},
					{"name":"Other income","kind":"income"}
				]
			}`,
		},
	})
}
