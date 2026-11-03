// Package testhelpers contains model helpers to be used in tests
package testhelpers

import (
	"github.com/logicorg/ivfs/pkg/model"
)

// Euro returns the Euro currency fields (except the ID)
func Euro() *model.Currency {
	return &model.Currency{
		Code:      "EUR",
		Name:      "Euro",
		Symbol:    "€",
		Precision: 2,
	}
}

// USDollar returns the USD currency (except the ID)
func USDollar() *model.Currency {
	return &model.Currency{
		Code:      "USD",
		Name:      "US Dollar",
		Symbol:    "$",
		Precision: 2,
	}
}

// BHD returns the BHD currency (except the ID). BHD is a currency with precision 3.
func BHD() *model.Currency {
	return &model.Currency{
		Code:      "BHD",
		Name:      "Bahraini Dinar",
		Symbol:    "BD",
		Precision: 3,
	}
}

// JapaneseYen returns the JPY currency (except the ID). JPY has precision 0 (no decimals).
func JapaneseYen() *model.Currency {
	return &model.Currency{
		Code:      "JPY",
		Name:      "Japanese Yen",
		Symbol:    "¥",
		Precision: 0,
	}
}
