package fx

import (
	"time"

	"github.com/shopspring/decimal"
)

// RegisterFXConversionToolInput mirrors RegisterFXConversionInput with
// IdempotencyKey dropped: the workflow fills a random one in when it is left
// empty (workflows.DefaultIdempotencyKey), so an MCP caller never needs to
// invent one. Field names must match RegisterFXConversionInput's JSON tags,
// since Temporal deserializes this value into that type on the worker side -
// a field missing here just deserializes to its zero value there.
type RegisterFXConversionToolInput struct {
	OccurredAt        time.Time `json:"occurredAt"`
	Description       *string   `json:"description,omitempty"`
	CounterpartyID    *int64    `json:"counterpartyId,omitempty"`
	ExternalID        *string   `json:"externalId,omitempty"`
	ProviderReference *string   `json:"providerReference,omitempty"`

	SourceLedgerAccountID         int64           `json:"sourceLedgerAccountId"`
	SourceClearingLedgerAccountID int64           `json:"sourceClearingLedgerAccountId"`
	SourceAmount                  decimal.Decimal `json:"sourceAmount"`

	DestinationLedgerAccountID         int64           `json:"destinationLedgerAccountId"`
	DestinationClearingLedgerAccountID int64           `json:"destinationClearingLedgerAccountId"`
	DestinationAmount                  decimal.Decimal `json:"destinationAmount"`

	// Fee is the FX fee charged at source.
	Fee *FXFeeToolInput `json:"fee,omitempty"`
}

// FXFeeToolInput mirrors FXFee without the "Expenses" account, which the
// workflow finds.
type FXFeeToolInput struct {
	CategoryID int64           `json:"categoryId" jsonschema:"the expense category the fee is booked under, such as Bank fees"`
	Amount     decimal.Decimal `json:"amount" jsonschema:"the fee, in the source currency and part of the source amount"`
}
