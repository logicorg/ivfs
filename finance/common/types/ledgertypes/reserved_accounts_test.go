package ledgertypes

import (
	"testing"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
)

func TestIsReservedLedgerAccount(t *testing.T) {
	tests := []struct {
		name    string
		account model.LedgerAccount
		want    bool
	}{
		{name: "opening balances", account: model.LedgerAccount{Name: "Opening balances", Kind: model.LedgerAccountEquity}, want: true},
		{name: "fx clearing", account: model.LedgerAccount{Name: "FX clearing", Kind: model.LedgerAccountAsset}, want: true},
		{name: "untracked accounts", account: model.LedgerAccount{Name: "Untracked accounts", Kind: model.LedgerAccountEquity}, want: true},
		{name: "uncategorized expense", account: model.LedgerAccount{Name: "Uncategorized", Kind: model.LedgerAccountExpense}, want: true},
		{name: "uncategorized income", account: model.LedgerAccount{Name: "Uncategorized", Kind: model.LedgerAccountIncome}, want: true},
		{name: "other case and spaces", account: model.LedgerAccount{Name: " fx CLEARING ", Kind: model.LedgerAccountAsset}, want: true},
		{name: "another kind", account: model.LedgerAccount{Name: "FX clearing", Kind: model.LedgerAccountEquity}},
		{name: "a child", account: model.LedgerAccount{Name: "Opening balances", Kind: model.LedgerAccountEquity, ParentID: ptr.NewPtr(int64(1))}},
		{name: "of a financial account", account: model.LedgerAccount{Name: "FX clearing", Kind: model.LedgerAccountAsset, FinancialAccountID: ptr.NewPtr(int64(1))}},
		{name: "an ordinary account", account: model.LedgerAccount{Name: "Groceries", Kind: model.LedgerAccountExpense}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsReservedLedgerAccount(&tc.account))
		})
	}
}

func TestIsUntrackedInstitution(t *testing.T) {
	tests := []struct {
		name        string
		institution string
		want        bool
	}{
		{name: "the name", institution: "Untracked", want: true},
		{name: "other case and spaces", institution: " untracked ", want: true},
		{name: "another institution", institution: "Revolut"},
		{name: "a longer name", institution: "Untracked accounts"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsUntrackedInstitution(tc.institution))
		})
	}
}

func TestIsUntrackedFinancialAccountName(t *testing.T) {
	tests := []struct {
		name    string
		account string
		want    bool
	}{
		{name: "the name", account: "Untracked accounts", want: true},
		{name: "other case and spaces", account: " UNTRACKED accounts ", want: true},
		{name: "another account", account: "Revolut EUR"},
		{name: "the institution's name", account: "Untracked"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsUntrackedFinancialAccountName(tc.account))
		})
	}
}
