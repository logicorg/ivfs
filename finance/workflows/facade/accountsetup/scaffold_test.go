package accountsetup

import (
	"testing"

	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestScaffold(t *testing.T) {
	tests := map[ast.AccountType]struct {
		want            []balanceSpec
		institutionKind string
	}{
		ast.AccountTypeBank: {
			want:            []balanceSpec{{"Cash", model.LedgerAccountAsset}},
			institutionKind: "bank",
		},
		ast.AccountTypeCard: {
			want:            []balanceSpec{{"Card balance", model.LedgerAccountLiability}},
			institutionKind: "card issuer",
		},
		ast.AccountTypeBrokerage: {
			want:            []balanceSpec{{"Cash", model.LedgerAccountAsset}, {"Investments", model.LedgerAccountAsset}},
			institutionKind: "broker",
		},
		ast.AccountTypeCash: {
			want:            []balanceSpec{{"Cash", model.LedgerAccountAsset}},
			institutionKind: "cash",
		},
	}

	require.Len(t, tests, len(ast.AccountTypes), "every account type has a case")
	for accountType, tc := range tests {
		t.Run(string(accountType), func(t *testing.T) {
			require.Equal(t, tc.want, scaffold(accountType))
			require.Equal(t, tc.institutionKind, institutionKind(accountType))
		})
	}
}

func TestSharedAccounts(t *testing.T) {
	require.Equal(t, []balanceSpec{
		{"Opening balances", model.LedgerAccountEquity},
		{"FX clearing", model.LedgerAccountAsset},
	}, sharedAccounts())
}

func TestInstitutionAndAccountName(t *testing.T) {
	tests := map[string]struct {
		input           ast.OpenAccountInput
		wantInstitution string
		wantAccount     string
	}{
		"a named account": {
			input:           ast.OpenAccountInput{Type: ast.AccountTypeBank, InstitutionName: "Revolut", AccountName: "Revolut Joint"},
			wantInstitution: "Revolut",
			wantAccount:     "Revolut Joint",
		},
		"an unnamed account takes the institution's name": {
			input:           ast.OpenAccountInput{Type: ast.AccountTypeBrokerage, InstitutionName: "IBKR"},
			wantInstitution: "IBKR",
			wantAccount:     "IBKR",
		},
		"an unnamed cash account is the wallet": {
			input:           ast.OpenAccountInput{Type: ast.AccountTypeCash},
			wantInstitution: "Physical cash",
			wantAccount:     "Wallet",
		},
		"a named cash account": {
			input:           ast.OpenAccountInput{Type: ast.AccountTypeCash, AccountName: "Travel cash"},
			wantInstitution: "Physical cash",
			wantAccount:     "Travel cash",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.wantInstitution, institutionName(tc.input))
			require.Equal(t, tc.wantAccount, accountName(tc.input))
		})
	}
}

func TestOpeningBalanceKey(t *testing.T) {
	require.Equal(t, "opening-balance:42", openingBalanceKey(42))
}

// TestOpeningBalancePostings checks that an opening balance is booked in the
// balance's natural direction against "Opening balances": a debit for money
// held in an asset, a credit for money owed on a liability, the other way
// round for a negative amount, and nothing for zero.
func TestOpeningBalancePostings(t *testing.T) {
	const balance, opening = 10, 20

	tests := map[string]struct {
		kind   model.LedgerAccountKind
		amount string
		want   []lt.PostingInput
	}{
		"money held in an asset": {
			kind:   model.LedgerAccountAsset,
			amount: "1200.50",
			want: []lt.PostingInput{
				{LedgerAccountID: balance, Amount: decimal.RequireFromString("1200.50"), Side: model.PostingDebit},
				{LedgerAccountID: opening, Amount: decimal.RequireFromString("1200.50"), Side: model.PostingCredit},
			},
		},
		"an overdrawn asset": {
			kind:   model.LedgerAccountAsset,
			amount: "-75",
			want: []lt.PostingInput{
				{LedgerAccountID: balance, Amount: decimal.NewFromInt(75), Side: model.PostingCredit},
				{LedgerAccountID: opening, Amount: decimal.NewFromInt(75), Side: model.PostingDebit},
			},
		},
		"money owed on a card": {
			kind:   model.LedgerAccountLiability,
			amount: "300",
			want: []lt.PostingInput{
				{LedgerAccountID: balance, Amount: decimal.NewFromInt(300), Side: model.PostingCredit},
				{LedgerAccountID: opening, Amount: decimal.NewFromInt(300), Side: model.PostingDebit},
			},
		},
		"a card in credit": {
			kind:   model.LedgerAccountLiability,
			amount: "-20",
			want: []lt.PostingInput{
				{LedgerAccountID: balance, Amount: decimal.NewFromInt(20), Side: model.PostingDebit},
				{LedgerAccountID: opening, Amount: decimal.NewFromInt(20), Side: model.PostingCredit},
			},
		},
		"zero": {
			kind:   model.LedgerAccountAsset,
			amount: "0.00",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := openingBalancePostings(balance, tc.kind, opening, decimal.RequireFromString(tc.amount))
			require.Len(t, got, len(tc.want))
			for i := range tc.want {
				require.Equal(t, tc.want[i].LedgerAccountID, got[i].LedgerAccountID)
				require.Equal(t, tc.want[i].Side, got[i].Side)
				require.Truef(t, tc.want[i].Amount.Equal(got[i].Amount), "amount %s, want %s", got[i].Amount, tc.want[i].Amount)
			}
		})
	}
}
