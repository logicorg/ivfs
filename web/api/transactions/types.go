package transactions

import (
	"time"

	"github.com/logicorg/ivfs/web/api/gateway"
)

// ExpenseForm is the answers to the form that records money spent from a
// balance on a category.
type ExpenseForm struct {
	IdempotencyKey          string         `json:"idempotencyKey" minLength:"1" doc:"one per transaction meant, such as one per opening of the form; a repeated key returns the entry first recorded with it"`
	BalanceLedgerAccountID  int64          `json:"balanceLedgerAccountId" minimum:"1" doc:"the balance the money leaves"`
	CategoryLedgerAccountID int64          `json:"categoryLedgerAccountId" minimum:"1" doc:"the expense category it is spent on, in the balance's currency"`
	Amount                  gateway.Amount `json:"amount" doc:"the amount spent, greater than zero"`
	OccurredAt              time.Time      `json:"occurredAt" doc:"the instant the money was spent"`
	Description             string         `json:"description" required:"false" doc:"a note on the entry"`
}

// IncomeForm is the answers to the form that records money earned on a
// category into a balance.
type IncomeForm struct {
	IdempotencyKey          string         `json:"idempotencyKey" minLength:"1" doc:"one per transaction meant, such as one per opening of the form; a repeated key returns the entry first recorded with it"`
	BalanceLedgerAccountID  int64          `json:"balanceLedgerAccountId" minimum:"1" doc:"the balance the money reaches"`
	CategoryLedgerAccountID int64          `json:"categoryLedgerAccountId" minimum:"1" doc:"the income category it is earned on, in the balance's currency"`
	Amount                  gateway.Amount `json:"amount" doc:"the amount earned, greater than zero"`
	OccurredAt              time.Time      `json:"occurredAt" doc:"the instant the money was earned"`
	Description             string         `json:"description" required:"false" doc:"a note on the entry"`
}

// TransferForm is the answers to the form that records money moved from one
// balance to another, such as a card paid off, possibly into another
// currency.
type TransferForm struct {
	IdempotencyKey      string         `json:"idempotencyKey" minLength:"1" doc:"one per transaction meant, such as one per opening of the form; a repeated key returns the entry first recorded with it"`
	FromLedgerAccountID int64          `json:"fromLedgerAccountId" minimum:"1" doc:"the balance the money leaves"`
	ToLedgerAccountID   int64          `json:"toLedgerAccountId" minimum:"1" doc:"the balance the money reaches"`
	Amount              gateway.Amount `json:"amount" doc:"the amount that leaves, greater than zero"`
	ToAmount            gateway.Amount `json:"toAmount" required:"false" doc:"the amount that arrives: required between currencies, else omitted or equal to amount"`
	OccurredAt          time.Time      `json:"occurredAt" doc:"the instant the money moved"`
	Description         string         `json:"description" required:"false" doc:"a note on the entry"`
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
