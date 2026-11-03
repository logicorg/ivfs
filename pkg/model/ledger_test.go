package model

import "testing"

func TestPostingSideOpposite(t *testing.T) {
	tests := []struct {
		name string
		in   PostingSide
		want PostingSide
	}{
		{name: "debit", in: PostingDebit, want: PostingCredit},
		{name: "credit", in: PostingCredit, want: PostingDebit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Opposite(); got != tt.want {
				t.Fatalf("Opposite() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsValidPostingSide(t *testing.T) {
	tests := map[string]struct {
		side PostingSide
		want bool
	}{
		"debit":   {side: PostingDebit, want: true},
		"credit":  {side: PostingCredit, want: true},
		"invalid": {side: PostingSide("invalid"), want: false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := IsValidPostingSide(test.side); got != test.want {
				t.Fatalf("IsValidPostingSide(%q) = %t, want %t", test.side, got, test.want)
			}
		})
	}
}

func TestIsValidJournalEntryStatus(t *testing.T) {
	tests := map[string]struct {
		status JournalEntryStatus
		want   bool
	}{
		"draft":   {status: JournalEntryDraft, want: true},
		"posted":  {status: JournalEntryPosted, want: true},
		"voided":  {status: JournalEntryStatus("voided"), want: false},
		"empty":   {status: JournalEntryStatus(""), want: false},
		"invalid": {status: JournalEntryStatus("invalid"), want: false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := IsValidJournalEntryStatus(test.status); got != test.want {
				t.Fatalf("IsValidJournalEntryStatus(%q) = %t, want %t", test.status, got, test.want)
			}
		})
	}
}

func TestLedgerAccountIsBrokerageInvestments(t *testing.T) {
	brokerage := int64(1)
	tests := map[string]struct {
		account LedgerAccount
		want    bool
	}{
		"a brokerage's Investments":       {account: LedgerAccount{Kind: LedgerAccountAsset, FinancialAccountID: &brokerage, Name: "Investments"}, want: true},
		"in any case":                     {account: LedgerAccount{Kind: LedgerAccountAsset, FinancialAccountID: &brokerage, Name: "investments"}, want: true},
		"a brokerage's Cash":              {account: LedgerAccount{Kind: LedgerAccountAsset, FinancialAccountID: &brokerage, Name: "Cash"}, want: false},
		"one outside a financial account": {account: LedgerAccount{Kind: LedgerAccountAsset, Name: "Investments"}, want: false},
		"a liability of that name":        {account: LedgerAccount{Kind: LedgerAccountLiability, FinancialAccountID: &brokerage, Name: "Investments"}, want: false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := test.account.IsBrokerageInvestments(); got != test.want {
				t.Fatalf("IsBrokerageInvestments() = %t, want %t", got, test.want)
			}
		})
	}
}
