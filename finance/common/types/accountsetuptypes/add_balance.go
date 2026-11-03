package accountsetuptypes

import (
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// AddBalanceInput adds one named balance, such as a savings vault, to an
// existing financial account (ADR 0006).
type AddBalanceInput struct {
	FinancialAccountID int64           `json:"financialAccountId" cue:">0" jsonschema:"the account to add the balance to, from list_accounts"`
	Name               string          `json:"name" cue:"!~\"^[[:space:]]*$\"" jsonschema:"the balance's name, such as Savings vault"`
	CurrencyCode       string          `json:"currencyCode" cue:"!~\"^[[:space:]]*$\"" jsonschema:"an ISO 4217 code, such as EUR"`
	OpeningBalance     decimal.Decimal `json:"openingBalance,omitzero" jsonschema:"the money in it today; nothing is posted if omitted"`
	// OpeningBalanceAt is when tracking starts; the time of the request if
	// omitted.
	OpeningBalanceAt time.Time `json:"openingBalanceAt,omitzero" jsonschema:"the instant the opening balance holds at, such as 2026-01-01T00:00:00Z; now if omitted"`
}

// Validate checks the input against its CUE constraints.
func (i *AddBalanceInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

// AddBalanceResult is the balance AddBalance added, created now or before,
// with the shared accounts of its currency.
type AddBalanceResult struct {
	Outcome         Outcome `json:"outcome"`
	LedgerAccountID int64   `json:"ledgerAccountId"`
	Balances
}
