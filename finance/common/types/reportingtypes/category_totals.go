package reportingtypes

import (
	"time"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// CategoryTotalsParams selects the period to total categories over. The range
// is half-open on the date the event occurred, as in ListJournalEntriesParams:
// From is included and To is not, so consecutive periods neither overlap nor
// leave a gap.
type CategoryTotalsParams struct {
	From time.Time `json:"from" jsonschema:"count entries that occurred at or after this RFC 3339 instant, such as 2026-01-01T00:00:00Z"`
	To   time.Time `json:"to" jsonschema:"count entries that occurred before this RFC 3339 instant, such as 2026-02-01T00:00:00Z"`
}

// Validate checks the parameters without the database.
func (p *CategoryTotalsParams) Validate() error {
	var missing []string
	if p.From.IsZero() {
		missing = append(missing, "from")
	}
	if p.To.IsZero() {
		missing = append(missing, "to")
	}
	if len(missing) > 0 {
		return internalerrors.ErrMissingRequiredParameter{Parameters: missing}
	}
	if !p.To.After(p.From) {
		return internalerrors.ErrInvalidParameter{Description: "the end of the period must be after its start"}
	}
	return nil
}

// CategoryTotal is what was earned in an income category, or spent in an
// expense category, in one currency during a period: credits less debits for
// income, debits less credits for an expense. A refund larger than the
// spending makes it negative. Without a CategoryID it is the uncategorized
// total of the kind and currency, so the totals of a kind and currency sum to
// what its "Income" or "Expenses" account took.
type CategoryTotal struct {
	CategoryID   *int64                  `json:"categoryId,omitempty"`
	Kind         model.LedgerAccountKind `json:"kind"`
	CurrencyID   int64                   `json:"currencyId"`
	CurrencyCode string                  `json:"currencyCode"`
	Total        decimal.Decimal         `json:"total"`
}

// CategoryTotalsResult is the total of each category and currency with a
// posted entry in the period, ordered by category id, the uncategorized totals
// last, and then by kind and currency. A category without an entry is left
// out, and a total is of the category itself, not of those under it.
type CategoryTotalsResult struct {
	Totals []CategoryTotal `json:"totals"`
}
