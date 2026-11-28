package transactions

import (
	"time"

	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// SplitForm is part of an amount booked on an expense category of its own: a
// line of an expense, or a charge of a transfer.
type SplitForm struct {
	CategoryID int64          `json:"categoryId" minimum:"1" doc:"the expense category it is booked under"`
	Amount     gateway.Amount `json:"amount" doc:"the part of the amount it takes, greater than zero"`
}

// splits are the forms as the workflows take them, nil for none.
func splits(forms []SplitForm) []tt.Split {
	if len(forms) == 0 {
		return nil
	}
	splits := make([]tt.Split, len(forms))
	for i, form := range forms {
		splits[i] = tt.Split{CategoryID: form.CategoryID, Amount: form.Amount.Decimal}
	}
	return splits
}

// ExpenseForm is the answers to the form that records money spent from a
// balance on a category.
type ExpenseForm struct {
	IdempotencyKey         string         `json:"idempotencyKey" minLength:"1" doc:"one per transaction meant, such as one per opening of the form; a repeated key returns the entry first recorded with it"`
	BalanceLedgerAccountID int64          `json:"balanceLedgerAccountId" minimum:"1" doc:"the balance the money leaves"`
	CategoryID             int64          `json:"categoryId" minimum:"1" required:"false" doc:"the expense category it is spent on; omitted when lines are given"`
	Amount                 gateway.Amount `json:"amount" doc:"the amount spent, greater than zero"`
	OccurredAt             time.Time      `json:"occurredAt" doc:"the instant the money was spent"`
	Description            string         `json:"description" required:"false" doc:"a note on the entry"`
	TagIDs                 []int64        `json:"tagIds" required:"false" doc:"the tags of the entry; a repeated id counts once"`
	Lines                  []SplitForm    `json:"lines" required:"false" doc:"what a payment spent on several things went on, adding up to amount, in place of categoryId"`
}

// IncomeForm is the answers to the form that records money earned on a
// category into a balance.
type IncomeForm struct {
	IdempotencyKey         string         `json:"idempotencyKey" minLength:"1" doc:"one per transaction meant, such as one per opening of the form; a repeated key returns the entry first recorded with it"`
	BalanceLedgerAccountID int64          `json:"balanceLedgerAccountId" minimum:"1" doc:"the balance the money reaches"`
	CategoryID             int64          `json:"categoryId" minimum:"1" doc:"the income category it is earned on"`
	Amount                 gateway.Amount `json:"amount" doc:"the amount earned, greater than zero"`
	OccurredAt             time.Time      `json:"occurredAt" doc:"the instant the money was earned"`
	Description            string         `json:"description" required:"false" doc:"a note on the entry"`
	TagIDs                 []int64        `json:"tagIds" required:"false" doc:"the tags of the entry; a repeated id counts once"`
}

// TransferForm is the answers to the form that records money moved from one
// balance to another, such as a card paid off, possibly into another
// currency.
type TransferForm struct {
	IdempotencyKey      string         `json:"idempotencyKey" minLength:"1" doc:"one per transaction meant, such as one per opening of the form; a repeated key returns the entry first recorded with it"`
	FromLedgerAccountID int64          `json:"fromLedgerAccountId" minimum:"1" doc:"the balance the money leaves"`
	ToLedgerAccountID   int64          `json:"toLedgerAccountId" minimum:"1" doc:"the balance the money reaches"`
	Amount              gateway.Amount `json:"amount" doc:"the amount that leaves, greater than zero"`
	ToAmount            gateway.Amount `json:"toAmount" required:"false" doc:"the amount that arrives: required between currencies, else omitted or equal to amount less the charges"`
	OccurredAt          time.Time      `json:"occurredAt" doc:"the instant the money moved"`
	Description         string         `json:"description" required:"false" doc:"a note on the entry"`
	TagIDs              []int64        `json:"tagIds" required:"false" doc:"the tags of the entry; a repeated id counts once"`
	Charges             []SplitForm    `json:"charges" required:"false" doc:"what moving the money cost, such as an instalment's interest, taken out of amount; at most one, the exchange fee, between currencies"`
}

// RecordedTransaction is the journal entry a transaction is recorded as,
// posted now or before with the same idempotency key.
type RecordedTransaction struct {
	JournalEntryID int64 `json:"journalEntryId"`
}

// RecordExpenseInput is the request of POST /api/transactions/expenses.
type RecordExpenseInput struct {
	Body ExpenseForm
}

// RecordIncomeInput is the request of POST /api/transactions/incomes.
type RecordIncomeInput struct {
	Body IncomeForm
}

// RecordTransferInput is the request of POST /api/transactions/transfers.
type RecordTransferInput struct {
	Body TransferForm
}

// RecordTransactionOutput is the response of every transaction endpoint.
type RecordTransactionOutput struct {
	Body RecordedTransaction
}
