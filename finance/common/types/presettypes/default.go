package presettypes

import (
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
)

// Default is the preset a new user is offered. It is a function,
// not a variable, so that no caller can change what the next one gets.
func Default() ApplyPresetInput {
	// Every precision is spelled out, so that a form shows the one it gets.
	return ApplyPresetInput{
		Currencies: []ct.InsertCurrencyParams{
			{Code: "EUR", Name: "Euro", Symbol: "€", Precision: ptr.NewPtr(2)},
			{Code: "USD", Name: "US Dollar", Symbol: "$", Precision: ptr.NewPtr(2)},
			{Code: "GBP", Name: "British Pound", Symbol: "£", Precision: ptr.NewPtr(2)},
			{Code: "CHF", Name: "Swiss Franc", Symbol: "CHF", Precision: ptr.NewPtr(2)},
			{Code: "JPY", Name: "Japanese Yen", Symbol: "¥", Precision: ptr.NewPtr(0)},
		},
		Tags: []tt.InsertTagParams{
			{Name: "vacation"},
			{Name: "reimbursable"},
			{Name: "tax-deductible"},
			{Name: "recurring"},
		},
		Categories: []PresetCategory{
			expense("Housing", "Rent or mortgage", "Utilities", "Maintenance"),
			expense("Food", "Groceries", "Eating out"),
			expense("Transport", "Public transport", "Fuel", "Vehicle"),
			expense("Leisure", "Subscriptions", "Travel", "Hobbies"),
			expense("Health"),
			expense("Shopping"),
			expense("Taxes"),
			expense("Fees"),
			income("Salary"),
			income("Interest"),
			income("Dividends"),
			income("Gifts"),
			income("Other income"),
		},
	}
}

func expense(name string, children ...string) PresetCategory {
	return category(model.LedgerAccountExpense, name, children)
}

func income(name string, children ...string) PresetCategory {
	return category(model.LedgerAccountIncome, name, children)
}

func category(kind model.LedgerAccountKind, name string, children []string) PresetCategory {
	c := PresetCategory{Name: name, Kind: kind}
	for _, child := range children {
		c.Children = append(c.Children, PresetCategory{Name: child})
	}
	return c
}
