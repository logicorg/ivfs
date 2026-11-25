package presettypes

import (
	"fmt"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestApplyPresetInput_Validate(t *testing.T) {
	euro := ct.InsertCurrencyParams{Code: "EUR", Name: "Euro", Symbol: "€"}
	food := PresetCategory{Name: "Food", Kind: model.LedgerAccountExpense}

	tests := map[string]struct {
		input     ApplyPresetInput
		wantError string
	}{
		"one of each": {
			input: ApplyPresetInput{
				Currencies: []ct.InsertCurrencyParams{euro},
				Tags:       []tt.InsertTagParams{{Name: "vacation"}},
				Categories: []PresetCategory{food},
			},
		},
		"only tags": {
			input: ApplyPresetInput{Tags: []tt.InsertTagParams{{Name: "vacation"}}},
		},
		"nothing": {
			input:     ApplyPresetInput{},
			wantError: "the preset is empty",
		},
		"an invalid currency": {
			input:     ApplyPresetInput{Currencies: []ct.InsertCurrencyParams{{Code: "euro", Name: "Euro"}}},
			wantError: `currency "euro": validation error`,
		},
		"a currency without a name": {
			input:     ApplyPresetInput{Currencies: []ct.InsertCurrencyParams{{Code: "EUR"}}},
			wantError: `currency "EUR": missing required parameters: Name`,
		},
		"a currency code twice": {
			input:     ApplyPresetInput{Currencies: []ct.InsertCurrencyParams{euro, {Code: "EUR", Name: "Other"}}},
			wantError: "currency EUR is given more than once",
		},
		"a currency name twice, in another case": {
			input:     ApplyPresetInput{Currencies: []ct.InsertCurrencyParams{euro, {Code: "EUX", Name: "EURO"}}},
			wantError: `currency name "EURO" is given more than once`,
		},
		"a blank tag": {
			input:     ApplyPresetInput{Tags: []tt.InsertTagParams{{Name: "  "}}},
			wantError: "missing required parameters: tag Name",
		},
		"a tag twice, in another case": {
			input:     ApplyPresetInput{Tags: []tt.InsertTagParams{{Name: "vacation"}, {Name: " Vacation "}}},
			wantError: `tag "Vacation" is given more than once`,
		},
		"a top-level category without a kind": {
			input:     ApplyPresetInput{Categories: []PresetCategory{{Name: "Food"}}},
			wantError: `category "Food" needs a kind, income or expense, not ""`,
		},
		"a top-level category of the wrong kind": {
			input:     ApplyPresetInput{Categories: []PresetCategory{{Name: "Cash", Kind: model.LedgerAccountAsset}}},
			wantError: `category "Cash" needs a kind, income or expense, not "asset"`,
		},
		"a blank top-level category": {
			input:     ApplyPresetInput{Categories: []PresetCategory{{Name: " ", Kind: model.LedgerAccountExpense}}},
			wantError: "missing required parameters: category Name",
		},
		"a top-level category twice": {
			input:     ApplyPresetInput{Categories: []PresetCategory{food, {Name: "food", Kind: model.LedgerAccountExpense}}},
			wantError: `expense category "food" is given more than once`,
		},
		"the same name as an income and an expense": {
			input: ApplyPresetInput{Categories: []PresetCategory{
				{Name: "Other", Kind: model.LedgerAccountExpense}, {Name: "Other", Kind: model.LedgerAccountIncome},
			}},
		},
		"a child with a kind": {
			input: ApplyPresetInput{Categories: []PresetCategory{
				{Name: "Food", Kind: model.LedgerAccountExpense, Children: []PresetCategory{{Name: "Groceries", Kind: model.LedgerAccountExpense}}},
			}},
			wantError: `category "Groceries" takes the kind of "Food": leave its kind out`,
		},
		"a blank child": {
			input: ApplyPresetInput{Categories: []PresetCategory{
				{Name: "Food", Kind: model.LedgerAccountExpense, Children: []PresetCategory{{Name: ""}}},
			}},
			wantError: "missing required parameters: category Name",
		},
		"a child twice": {
			input: ApplyPresetInput{Categories: []PresetCategory{
				{Name: "Food", Kind: model.LedgerAccountExpense, Children: []PresetCategory{{Name: "Groceries"}, {Name: "GROCERIES"}}},
			}},
			wantError: `category "GROCERIES" is given more than once under "Food"`,
		},
		"the same child under two parents": {
			input: ApplyPresetInput{Categories: []PresetCategory{
				{Name: "Food", Kind: model.LedgerAccountExpense, Children: []PresetCategory{{Name: "Other"}}},
				{Name: "Transport", Kind: model.LedgerAccountExpense, Children: []PresetCategory{{Name: "Other"}}},
			}},
		},
		"a grandchild": {
			input: ApplyPresetInput{Categories: []PresetCategory{
				{Name: "Food", Kind: model.LedgerAccountExpense, Children: []PresetCategory{
					{Name: "Groceries", Children: []PresetCategory{{Name: "Organic"}}},
				}},
			}},
		},
		"a great-grandchild": {
			input: ApplyPresetInput{Categories: []PresetCategory{
				{Name: "Food", Kind: model.LedgerAccountExpense, Children: []PresetCategory{
					{Name: "Groceries", Children: []PresetCategory{{Name: "Organic", Children: []PresetCategory{{Name: "Fruit"}}}}},
				}},
			}},
			wantError: `category "Organic" is 2 levels deep and cannot have children (at most 2 parent links)`,
		},
		"100 tags": {
			input: ApplyPresetInput{Tags: tags(100)},
		},
		"101 tags": {
			input:     ApplyPresetInput{Tags: tags(101)},
			wantError: "101 tags is too many: a preset holds at most 100",
		},
		"101 currencies": {
			input:     ApplyPresetInput{Currencies: make([]ct.InsertCurrencyParams, 101)},
			wantError: "101 currencies is too many",
		},
		"101 categories counting children": {
			input: ApplyPresetInput{Categories: []PresetCategory{
				{Name: "Food", Kind: model.LedgerAccountExpense, Children: categories(100)},
			}},
			wantError: "101 categories is too many",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func tags(n int) []tt.InsertTagParams {
	out := make([]tt.InsertTagParams, n)
	for i := range out {
		out[i] = tt.InsertTagParams{Name: fmt.Sprintf("tag %d", i)}
	}
	return out
}

func categories(n int) []PresetCategory {
	out := make([]PresetCategory, n)
	for i := range out {
		out[i] = PresetCategory{Name: fmt.Sprintf("category %d", i)}
	}
	return out
}
