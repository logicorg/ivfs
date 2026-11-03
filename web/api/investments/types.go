package investments

import (
	"time"

	"github.com/logicorg/ivfs/web/api/gateway"
)

// BuyForm is the answers to the form that records units of an asset bought
// with a brokerage account's cash.
type BuyForm struct {
	IdempotencyKey     string         `json:"idempotencyKey" minLength:"1" doc:"one per buy meant, such as one per opening of the form; a repeated key returns the buy first recorded with it"`
	FinancialAccountID int64          `json:"financialAccountId" minimum:"1" doc:"the brokerage account bought in"`
	AssetID            int64          `json:"assetId" minimum:"1" doc:"the asset bought"`
	Units              gateway.Amount `json:"units" doc:"the units bought, greater than zero"`
	Amount             gateway.Amount `json:"amount" doc:"the price paid for the units, before the fee, greater than zero"`
	Fee                gateway.Amount `json:"fee" required:"false" doc:"the broker's fee, paid on top of the amount and part of the cost; zero if omitted"`
	CurrencyCode       string         `json:"currencyCode" minLength:"1" doc:"the currency paid in, such as EUR, in which the account has its Cash and Investments balances"`
	OccurredAt         time.Time      `json:"occurredAt" doc:"the instant of the buy"`
	Description        string         `json:"description" required:"false" doc:"a note on the entry"`
}

// SellForm is the answers to the form that records units of an asset sold for
// a brokerage account's cash.
type SellForm struct {
	IdempotencyKey     string         `json:"idempotencyKey" minLength:"1" doc:"one per sale meant, such as one per opening of the form; a repeated key returns the sale first recorded with it"`
	FinancialAccountID int64          `json:"financialAccountId" minimum:"1" doc:"the brokerage account sold in"`
	AssetID            int64          `json:"assetId" minimum:"1" doc:"the asset sold"`
	Units              gateway.Amount `json:"units" doc:"the units sold, greater than zero"`
	Amount             gateway.Amount `json:"amount" doc:"the price received for the units, before the fee, greater than zero"`
	Fee                gateway.Amount `json:"fee" required:"false" doc:"the broker's fee, taken from the proceeds and less than the amount; zero if omitted"`
	CurrencyCode       string         `json:"currencyCode" minLength:"1" doc:"the currency received, such as EUR, in which the account has its Cash and Investments balances"`
	OccurredAt         time.Time      `json:"occurredAt" doc:"the instant of the sale"`
	Description        string         `json:"description" required:"false" doc:"a note on the entry"`
}

// VestForm is the answers to the form that records units of an asset received
// as compensation, such as RSUs, in a brokerage account.
type VestForm struct {
	IdempotencyKey                string         `json:"idempotencyKey" minLength:"1" doc:"one per vest meant, such as one per opening of the form; a repeated key returns the vest first recorded with it"`
	FinancialAccountID            int64          `json:"financialAccountId" minimum:"1" doc:"the brokerage account the units are delivered to"`
	AssetID                       int64          `json:"assetId" minimum:"1" doc:"the asset vested"`
	UnitsVested                   gateway.Amount `json:"unitsVested" doc:"every unit vested, those withheld included, greater than zero"`
	UnitsWithheld                 gateway.Amount `json:"unitsWithheld" required:"false" doc:"the units the employer kept for tax, fewer than those vested; zero if omitted"`
	FairMarketValue               gateway.Amount `json:"fairMarketValue" doc:"the value of one unit at the vest, greater than zero"`
	CurrencyCode                  string         `json:"currencyCode" minLength:"1" doc:"the currency of the value, such as EUR, in which the account has its Cash and Investments balances and the categories are"`
	IncomeCategoryLedgerAccountID int64          `json:"incomeCategoryLedgerAccountId" minimum:"1" doc:"the income category credited with the value of the units vested"`
	TaxCategoryLedgerAccountID    int64          `json:"taxCategoryLedgerAccountId" required:"false" minimum:"1" doc:"the expense category debited with the value of the units withheld; required when units are withheld"`
	OccurredAt                    time.Time      `json:"occurredAt" doc:"the instant of the vest"`
	Description                   string         `json:"description" required:"false" doc:"a note on the entry"`
}

// RecordedTrade is the journal entry and the position movement a buy, a sale or
// a vest is recorded as, now or before with the same idempotency key.
type RecordedTrade struct {
	JournalEntryID     int64 `json:"journalEntryId"`
	PositionMovementID int64 `json:"positionMovementId"`
	// Warnings are the positions the trade leaves negative at some point in
	// time, such as by selling more units than are held. The trade is
	// recorded all the same (ADR 0010).
	Warnings []gateway.NegativeHolding `json:"warnings,omitempty"`
}

// BuyAssetInput is the request of POST /api/investments/buys.
type BuyAssetInput struct {
	Body BuyForm
}

// SellAssetInput is the request of POST /api/investments/sells.
type SellAssetInput struct {
	Body SellForm
}

// VestAssetInput is the request of POST /api/investments/vests.
type VestAssetInput struct {
	Body VestForm
}

// RecordTradeOutput is the response of every investment endpoint.
type RecordTradeOutput struct {
	Body RecordedTrade
}
