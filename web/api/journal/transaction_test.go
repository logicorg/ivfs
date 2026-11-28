package journal

import (
	"testing"
	"time"

	fxt "github.com/logicorg/ivfs/finance/common/types/fxtypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/gateway"
	"github.com/logicorg/ivfs/web/api/transactions"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// TestTransactionOf checks that an entry posted by a transaction is read back
// as the transaction, its splits in line order, and that one no transaction
// posts is not.
func TestTransactionOf(t *testing.T) {
	day := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	cash := &model.LedgerAccount{ID: 100, Name: "Cash", Kind: model.LedgerAccountAsset, CurrencyID: 1, FinancialAccountID: ptr.Int64(10)}
	cashUSD := &model.LedgerAccount{ID: 103, Name: "Cash", Kind: model.LedgerAccountAsset, CurrencyID: 2, FinancialAccountID: ptr.Int64(10)}
	principal := &model.LedgerAccount{ID: 104, Name: "Principal", Kind: model.LedgerAccountLiability, CurrencyID: 1, FinancialAccountID: ptr.Int64(11)}
	investments := &model.LedgerAccount{ID: 105, Name: "Investments", Kind: model.LedgerAccountAsset, CurrencyID: 1, FinancialAccountID: ptr.Int64(12)}
	expenses := &model.LedgerAccount{ID: 200, Name: "Expenses", Kind: model.LedgerAccountExpense, CurrencyID: 1}
	income := &model.LedgerAccount{ID: 300, Name: "Income", Kind: model.LedgerAccountIncome, CurrencyID: 1}
	openingBalances := &model.LedgerAccount{ID: 400, Name: "Opening balances", Kind: model.LedgerAccountEquity, CurrencyID: 1}
	clearing := &model.LedgerAccount{ID: 500, Name: "FX clearing", Kind: model.LedgerAccountAsset, CurrencyID: 1}
	clearingUSD := &model.LedgerAccount{ID: 501, Name: "FX clearing", Kind: model.LedgerAccountAsset, CurrencyID: 2}

	type line struct {
		account  *model.LedgerAccount
		side     model.PostingSide
		amount   string
		category int64
	}
	debit := func(account *model.LedgerAccount, amount string, category int64) line {
		return line{account: account, side: model.PostingDebit, amount: amount, category: category}
	}
	credit := func(account *model.LedgerAccount, amount string, category int64) line {
		return line{account: account, side: model.PostingCredit, amount: amount, category: category}
	}
	amount := func(value string) gateway.Amount { return gateway.NewAmount(decimal.RequireFromString(value)) }

	tests := []struct {
		name     string
		draft    bool
		fx       bool
		postings []line
		want     *EntryTransaction
	}{
		{
			name:     "an expense",
			postings: []line{debit(expenses, "12.50", 20), credit(cash, "12.50", 0)},
			want:     &EntryTransaction{Expense: &transactions.Expense{BalanceLedgerAccountID: 100, CategoryID: 20, Amount: amount("12.50"), OccurredAt: day}},
		},
		{
			name:     "an expense in lines",
			postings: []line{debit(expenses, "10", 20), debit(expenses, "2.50", 21), credit(cash, "12.50", 0)},
			want: &EntryTransaction{Expense: &transactions.Expense{
				BalanceLedgerAccountID: 100, Amount: amount("12.50"), OccurredAt: day,
				Lines: []transactions.SplitForm{{CategoryID: 20, Amount: amount("10")}, {CategoryID: 21, Amount: amount("2.50")}},
			}},
		},
		{
			name:     "an income",
			postings: []line{debit(cash, "2500", 0), credit(income, "2500", 30)},
			want:     &EntryTransaction{Income: &transactions.Income{BalanceLedgerAccountID: 100, CategoryID: 30, Amount: amount("2500"), OccurredAt: day}},
		},
		{
			name:     "a transfer",
			postings: []line{debit(principal, "300", 0), credit(cash, "300", 0)},
			want:     &EntryTransaction{Transfer: &transactions.Transfer{FromLedgerAccountID: 100, ToLedgerAccountID: 104, Amount: amount("300"), OccurredAt: day}},
		},
		{
			name:     "an instalment, its interest a charge",
			postings: []line{debit(principal, "400", 0), debit(expenses, "800", 22), credit(cash, "1200", 0)},
			want: &EntryTransaction{Transfer: &transactions.Transfer{
				FromLedgerAccountID: 100, ToLedgerAccountID: 104, Amount: amount("1200"), OccurredAt: day,
				Charges: []transactions.SplitForm{{CategoryID: 22, Amount: amount("800")}},
			}},
		},
		{
			name: "an exchange with a fee",
			fx:   true,
			postings: []line{
				debit(clearing, "299", 0), debit(expenses, "1.50", 23), credit(cash, "300.50", 0),
				debit(cashUSD, "324.15", 0), credit(clearingUSD, "324.15", 0),
			},
			want: &EntryTransaction{Transfer: &transactions.Transfer{
				FromLedgerAccountID: 100, ToLedgerAccountID: 103, Amount: amount("300.50"), ToAmount: amount("324.15"), OccurredAt: day,
				Charges: []transactions.SplitForm{{CategoryID: 23, Amount: amount("1.50")}},
			}},
		},
		{name: "a draft", draft: true, postings: []line{debit(expenses, "12.50", 20), credit(cash, "12.50", 0)}},
		{name: "a buy", postings: []line{debit(investments, "100", 0), credit(cash, "100", 0)}},
		{name: "an opening balance", postings: []line{debit(cash, "100", 0), credit(openingBalances, "100", 0)}},
		{name: "a refund", postings: []line{debit(cash, "12.50", 0), credit(expenses, "12.50", 20)}},
		{name: "an uncategorized expense", postings: []line{debit(expenses, "12.50", 0), credit(cash, "12.50", 0)}},
		{name: "an income and an expense", postings: []line{debit(cash, "90", 0), debit(expenses, "10", 20), credit(income, "100", 30)}},
		{name: "a clearing account outside an exchange", postings: []line{debit(clearing, "100", 0), credit(cash, "100", 0)}},
		{
			name: "an exchange with two fees",
			fx:   true,
			postings: []line{
				debit(clearing, "298", 0), debit(expenses, "1.50", 23), debit(expenses, "1", 24), credit(cash, "300.50", 0),
				debit(cashUSD, "324.15", 0), credit(clearingUSD, "324.15", 0),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry := &model.JournalEntry{ID: 41, OccurredAt: day, Status: model.JournalEntryPosted}
			if tc.draft {
				entry.Status = model.JournalEntryDraft
			}
			if tc.fx {
				entry.Metadata = map[string]string{"type": fxt.FXConversionEntryType}
			}
			for i, line := range tc.postings {
				posting := model.Posting{
					LineNumber: i + 1, LedgerAccountID: line.account.ID, LedgerAccount: line.account,
					Amount: decimal.RequireFromString(line.amount), Side: line.side,
				}
				if line.category != 0 {
					posting.CategoryID = ptr.Int64(line.category)
				}
				entry.Postings = append(entry.Postings, posting)
			}

			require.Equal(t, tc.want, transactionOf(entry))
		})
	}
}

// TestTransactionOf_Details checks that a transaction read back keeps the
// entry's note and tags.
func TestTransactionOf_Details(t *testing.T) {
	day := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	entry := &model.JournalEntry{
		ID: 41, OccurredAt: day, Status: model.JournalEntryPosted, Description: ptr.NewPtr("Esselunga"),
		Tags: []model.Tag{{ID: 3, Name: "home"}, {ID: 5, Name: "work"}},
		Postings: []model.Posting{
			{
				LineNumber: 1, LedgerAccountID: 200, Amount: decimal.RequireFromString("12.50"), Side: model.PostingDebit, CategoryID: ptr.Int64(20),
				LedgerAccount: &model.LedgerAccount{ID: 200, Name: "Expenses", Kind: model.LedgerAccountExpense, CurrencyID: 1},
			},
			{
				LineNumber: 2, LedgerAccountID: 100, Amount: decimal.RequireFromString("12.50"), Side: model.PostingCredit,
				LedgerAccount: &model.LedgerAccount{ID: 100, Name: "Cash", Kind: model.LedgerAccountAsset, CurrencyID: 1, FinancialAccountID: ptr.Int64(10)},
			},
		},
	}

	require.Equal(t, &EntryTransaction{Expense: &transactions.Expense{
		BalanceLedgerAccountID: 100, CategoryID: 20, Amount: gateway.NewAmount(decimal.RequireFromString("12.50")),
		OccurredAt: day, Description: "Esselunga", TagIDs: []int64{3, 5},
	}}, transactionOf(entry))
}
