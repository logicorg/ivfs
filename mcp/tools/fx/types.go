package fx

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
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

	// Fee is the FX fee charged at source. FXFee itself has no relation
	// fields, so it is reused as-is rather than mirrored.
	Fee *lt.FXFee `json:"fee,omitempty"`
}
