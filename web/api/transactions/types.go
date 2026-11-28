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

// Expense is money spent from a balance on a category, as it is recorded or
// as an entry is corrected to.
type Expense struct {
	BalanceLedgerAccountID int64          `json:"balanceLedgerAccountId" minimum:"1" doc:"the balance the money leaves"`
	CategoryID             int64          `json:"categoryId" minimum:"1" required:"false" doc:"the expense category it is spent on; omitted when lines are given"`
	Amount                 gateway.Amount `json:"amount" doc:"the amount spent, greater than zero"`
	OccurredAt             time.Time      `json:"occurredAt" doc:"the instant the money was spent"`
	Description            string         `json:"description" required:"false" doc:"a note on the entry"`
	TagIDs                 []int64        `json:"tagIds" required:"false" doc:"the tags of the entry; a repeated id counts once"`
	Lines                  []SplitForm    `json:"lines" required:"false" doc:"what a payment spent on several things went on, adding up to amount, in place of categoryId"`
}

// Income is money earned on a category into a balance, as it is recorded or
// as an entry is corrected to.
type Income struct {
	BalanceLedgerAccountID int64          `json:"balanceLedgerAccountId" minimum:"1" doc:"the balance the money reaches"`
	CategoryID             int64          `json:"categoryId" minimum:"1" doc:"the income category it is earned on"`
	Amount                 gateway.Amount `json:"amount" doc:"the amount earned, greater than zero"`
	OccurredAt             time.Time      `json:"occurredAt" doc:"the instant the money was earned"`
	Description            string         `json:"description" required:"false" doc:"a note on the entry"`
	TagIDs                 []int64        `json:"tagIds" required:"false" doc:"the tags of the entry; a repeated id counts once"`
}

// Transfer is money moved from one balance to another, such as a card paid
// off, possibly into another currency, as it is recorded or as an entry is
// corrected to.
type Transfer struct {
	FromLedgerAccountID int64          `json:"fromLedgerAccountId" minimum:"1" doc:"the balance the money leaves"`
	ToLedgerAccountID   int64          `json:"toLedgerAccountId" minimum:"1" doc:"the balance the money reaches"`
	Amount              gateway.Amount `json:"amount" doc:"the amount that leaves, greater than zero"`
	ToAmount            gateway.Amount `json:"toAmount" required:"false" doc:"the amount that arrives: required between currencies, else omitted or equal to amount less the charges"`
	OccurredAt          time.Time      `json:"occurredAt" doc:"the instant the money moved"`
	Description         string         `json:"description" required:"false" doc:"a note on the entry"`
	TagIDs              []int64        `json:"tagIds" required:"false" doc:"the tags of the entry; a repeated id counts once"`
	Charges             []SplitForm    `json:"charges" required:"false" doc:"what moving the money cost, such as an instalment's interest, taken out of amount; at most one, the exchange fee, between currencies"`
}

// ExpenseForm is the answers to the form that records an expense.
type ExpenseForm struct {
	IdempotencyKey string `json:"idempotencyKey" minLength:"1" doc:"one per transaction meant, such as one per opening of the form; a repeated key returns the entry first recorded with it"`
	Expense
}

// IncomeForm is the answers to the form that records an income.
type IncomeForm struct {
	IdempotencyKey string `json:"idempotencyKey" minLength:"1" doc:"one per transaction meant, such as one per opening of the form; a repeated key returns the entry first recorded with it"`
	Income
}

// TransferForm is the answers to the form that records a transfer.
type TransferForm struct {
	IdempotencyKey string `json:"idempotencyKey" minLength:"1" doc:"one per transaction meant, such as one per opening of the form; a repeated key returns the entry first recorded with it"`
	Transfer
}

// RecordedTransaction is the journal entry a transaction is recorded as,
// posted now or before with the same idempotency key.
type RecordedTransaction struct {
	JournalEntryID int64 `json:"journalEntryId"`
}

// UpdatedTransaction is the journal entry a correction replaced, which keeps
// its id.
type UpdatedTransaction struct {
	JournalEntryID int64 `json:"journalEntryId"`
	// Warnings are the holdings left negative at some point in time, when the
	// entry was a trade whose units the correction removes. It is made all the
	// same.
	Warnings []gateway.NegativeHolding `json:"warnings,omitempty"`
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

// RecordTransactionOutput is the response of every endpoint that records a
// transaction.
type RecordTransactionOutput struct {
	Body RecordedTransaction
}

// UpdateExpenseInput is the request of PUT /api/transactions/expenses/{id}.
type UpdateExpenseInput struct {
	ID   int64 `path:"id" minimum:"1" doc:"the journal entry to correct"`
	Body Expense
}

// UpdateIncomeInput is the request of PUT /api/transactions/incomes/{id}.
type UpdateIncomeInput struct {
	ID   int64 `path:"id" minimum:"1" doc:"the journal entry to correct"`
	Body Income
}

// UpdateTransferInput is the request of PUT /api/transactions/transfers/{id}.
type UpdateTransferInput struct {
	ID   int64 `path:"id" minimum:"1" doc:"the journal entry to correct"`
	Body Transfer
}

// UpdateTransactionOutput is the response of every endpoint that corrects an
// entry to a transaction.
type UpdateTransactionOutput struct {
	Body UpdatedTransaction
}

// updated is an update's result as the API returns it.
func updated(result *tt.UpdateResult) *UpdateTransactionOutput {
	return &UpdateTransactionOutput{Body: UpdatedTransaction{
		JournalEntryID: result.JournalEntryID,
		Warnings:       gateway.NewNegativeHoldings(result.Warnings),
	}}
}
