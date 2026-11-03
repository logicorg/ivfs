package ledgertypes

import (
	"fmt"
	"time"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// FXConversionEntryType is the value stored under the "type" metadata key of
// every journal entry created by RegisterFXConversion.
const FXConversionEntryType = "fx_conversion"

// FXClearingAccountName is the name of the shared asset account, one per
// currency and of no financial account, that a conversion balances through
// (ADR 0006).
const FXClearingAccountName = "FX clearing"

// FXFee is a fee charged in the source currency of a conversion. It is posted
// as its own expense line and is never folded into the exchange rate.
type FXFee struct {
	LedgerAccountID int64           `json:"ledgerAccountId"`
	Amount          decimal.Decimal `json:"amount"`
}

// RegisterFXConversionInput converts money between two currencies as one
// balanced journal entry. Both amounts are what the provider actually settled;
// the effective rate is derived from them and stored as entry metadata, so no
// rounding difference can arise.
//
// The source amount includes the fee. Each currency balances through its own
// clearing account, which is how the entry stays balanced per currency.
type RegisterFXConversionInput struct {
	IdempotencyKey    string    `json:"idempotencyKey"`
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
	Fee *FXFee `json:"fee,omitempty"`
}

type RegisterFXConversionResult struct {
	JournalEntryID int64 `json:"journalEntryId"`
}

// AccountIDs returns the ledger accounts the conversion touches.
func (i *RegisterFXConversionInput) AccountIDs() []int64 {
	ids := []int64{
		i.SourceLedgerAccountID,
		i.SourceClearingLedgerAccountID,
		i.DestinationLedgerAccountID,
		i.DestinationClearingLedgerAccountID,
	}
	if i.Fee != nil {
		ids = append(ids, i.Fee.LedgerAccountID)
	}
	return ids
}

// NetSourceAmount is the source amount that reaches the source clearing
// account, that is the source amount minus the fee.
func (i *RegisterFXConversionInput) NetSourceAmount() decimal.Decimal {
	if i.Fee == nil {
		return i.SourceAmount
	}
	return i.SourceAmount.Sub(i.Fee.Amount)
}

// Validate checks the parts of the command that do not need the database.
func (i *RegisterFXConversionInput) Validate() error {
	var missing []string
	if i.IdempotencyKey == "" {
		missing = append(missing, "IdempotencyKey")
	}
	if i.OccurredAt.IsZero() {
		missing = append(missing, "OccurredAt")
	}
	if len(missing) > 0 {
		return internalerrors.ErrMissingRequiredParameter{Parameters: missing}
	}

	seen := make(map[int64]struct{})
	for _, id := range i.AccountIDs() {
		if id <= 0 {
			return internalerrors.ErrInvalidParameter{Description: "ledger account IDs must be positive"}
		}
		if _, ok := seen[id]; ok {
			return internalerrors.ErrInvalidParameter{Description: "ledger accounts of a conversion must all be different"}
		}
		seen[id] = struct{}{}
	}

	if !i.SourceAmount.IsPositive() || !i.DestinationAmount.IsPositive() {
		return internalerrors.ErrInvalidParameter{Description: "source and destination amounts must be positive"}
	}
	if i.Fee != nil {
		if !i.Fee.Amount.IsPositive() {
			return internalerrors.ErrInvalidParameter{Description: "fee amount must be positive"}
		}
		if !i.NetSourceAmount().IsPositive() {
			return internalerrors.ErrInvalidParameter{Description: "fee must be smaller than the source amount"}
		}
	}
	return nil
}

// ToJournalEntry builds the generic balanced entry for the conversion:
//
//	source currency:      debit source clearing (net), debit fee, credit source
//	destination currency: debit destination, credit destination clearing
func (i *RegisterFXConversionInput) ToJournalEntry() PostJournalEntryInput {
	postings := []PostingInput{
		{LedgerAccountID: i.SourceClearingLedgerAccountID, Amount: i.NetSourceAmount(), Side: model.PostingDebit},
	}
	if i.Fee != nil {
		postings = append(postings, PostingInput{LedgerAccountID: i.Fee.LedgerAccountID, Amount: i.Fee.Amount, Side: model.PostingDebit})
	}
	postings = append(postings,
		PostingInput{LedgerAccountID: i.SourceLedgerAccountID, Amount: i.SourceAmount, Side: model.PostingCredit},
		PostingInput{LedgerAccountID: i.DestinationLedgerAccountID, Amount: i.DestinationAmount, Side: model.PostingDebit},
		PostingInput{LedgerAccountID: i.DestinationClearingLedgerAccountID, Amount: i.DestinationAmount, Side: model.PostingCredit},
	)

	// The rate is kept as the fraction of the two settled amounts because a
	// division such as 90/99 has no exact decimal form.
	metadata := map[string]string{
		"type":           FXConversionEntryType,
		"effective_rate": fmt.Sprintf("%s/%s", i.DestinationAmount, i.NetSourceAmount()),
	}
	if i.ProviderReference != nil {
		metadata["provider_reference"] = *i.ProviderReference
	}

	return PostJournalEntryInput{
		IdempotencyKey: i.IdempotencyKey,
		OccurredAt:     i.OccurredAt,
		Description:    i.Description,
		CounterpartyID: i.CounterpartyID,
		ExternalID:     i.ExternalID,
		Metadata:       metadata,
		Postings:       postings,
	}
}
