package accountsetuptypes

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// OpenAccountInput holds the answers to the form that opens an account
// (ADR 0006). Everything else, such as the balances' names and the
// institution's kind, is derived from them.
type OpenAccountInput struct {
	Type AccountType `json:"type" jsonschema:"what the account holds: bank for money, card for a credit card's debt, brokerage for assets and money, cash for a wallet"`
	// InstitutionName is required, except for a cash account, which always
	// belongs to the institution "Physical cash".
	InstitutionName string `json:"institutionName,omitempty" jsonschema:"the institution that holds the account, such as Revolut; left out for a cash account"`
	// CreateInstitution confirms that a missing institution is to be created.
	CreateInstitution bool   `json:"createInstitution,omitempty" jsonschema:"create the institution if it does not exist; set only after the person has confirmed its name"`
	AccountName       string `json:"accountName,omitempty" jsonschema:"the account's name; the institution's name, or Wallet for cash, if omitted"`
	// Currencies are the currencies the account holds, each with its
	// optional opening balance; at least one.
	Currencies []CurrencyBalance `json:"currencies" jsonschema:"the currencies the account holds, at least one"`
	// OpeningBalancesAt is when tracking starts; the time of the request if
	// omitted.
	OpeningBalancesAt time.Time `json:"openingBalancesAt,omitzero" jsonschema:"the instant the opening balances hold at, such as 2026-01-01T00:00:00Z; now if omitted"`
}

// CurrencyBalance is one currency of an account, with its opening balance.
type CurrencyBalance struct {
	Code string `json:"code" cue:"!~\"^[[:space:]]*$\"" jsonschema:"an ISO 4217 code, such as EUR"`
	// OpeningBalance is in the balance's natural direction: money held for
	// an asset, money owed for a card. Zero posts nothing.
	OpeningBalance decimal.Decimal `json:"openingBalance,omitzero" jsonschema:"the balance today: money held, or owed on a card; negative for an overdraft"`
}

// Validate checks the input against its CUE constraints, and the rules CUE
// cannot express with a useful message: a known type, at least one currency,
// none given twice, and the institution named for every type but cash.
func (i *OpenAccountInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}

	if !slices.Contains(AccountTypes, i.Type) {
		return errs.ValidationError{Err: fmt.Errorf("unknown account type %q (must be one of %q)", i.Type, AccountTypes)}
	}
	if len(i.Currencies) == 0 {
		return errs.ValidationError{Err: fmt.Errorf("at least one currency is required")}
	}

	named := strings.TrimSpace(i.InstitutionName) != ""
	switch {
	case i.Type == AccountTypeCash && named:
		return errs.ValidationError{Err: fmt.Errorf("a cash account belongs to %q; leave the institution out", PhysicalCashInstitution)}
	case i.Type != AccountTypeCash && !named:
		return errs.ValidationError{Err: fmt.Errorf("the institution is required for a %s account", i.Type)}
	}

	seen := make(map[string]bool, len(i.Currencies))
	for _, currency := range i.Currencies {
		code := strings.ToUpper(strings.TrimSpace(currency.Code))
		if seen[code] {
			return errs.ValidationError{Err: fmt.Errorf("currency %s is given more than once", code)}
		}
		seen[code] = true
	}
	return nil
}

// OpenAccountResult is what OpenAccount did. For OutcomeInstitutionNotFound
// only Outcome and InstitutionName are set.
type OpenAccountResult struct {
	Outcome              Outcome `json:"outcome"`
	InstitutionID        int64   `json:"institutionId,omitempty"`
	InstitutionName      string  `json:"institutionName"`
	FinancialAccountID   int64   `json:"financialAccountId,omitempty"`
	FinancialAccountName string  `json:"financialAccountName,omitempty"`
	Balances
}
