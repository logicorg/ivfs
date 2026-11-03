package ledgertypes

import (
	"fmt"
	"time"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// NetWorthSeriesMaxRows bounds the rows of a series, Balances and Units
// together, so the result stays well under Temporal's payload limit: 10,000
// rows is under 1 MB (ADR 0015).
const NetWorthSeriesMaxRows = 10_000

// NetWorthSeriesParams selects the days of a series. Both are days, From and
// To included; the time of day is ignored.
type NetWorthSeriesParams struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// Validate checks the parameters without the database.
func (p *NetWorthSeriesParams) Validate() error {
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
	if p.To.Before(p.From) {
		return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the range ends on %s, before it starts on %s", p.To.Format(time.DateOnly), p.From.Format(time.DateOnly))}
	}
	return nil
}

// NetWorthBalance is the asset and the liability balances in one currency at
// the end of a day, each in its natural direction. The asset sum leaves out
// brokerages' "Investments" balances.
type NetWorthBalance struct {
	Date        time.Time       `json:"date"`
	CurrencyID  int64           `json:"currencyId"`
	Assets      decimal.Decimal `json:"assets"`
	Liabilities decimal.Decimal `json:"liabilities"`
}

// NetWorthUnits is the units held of one asset at the end of a day, across
// every financial account. A negative holding stays negative (ADR 0010).
type NetWorthUnits struct {
	Date    time.Time       `json:"date"`
	AssetID int64           `json:"assetId"`
	Units   decimal.Decimal `json:"units"`
}

// NetWorthSeriesResult is a sparse series, oldest first (ADR 0015). Each
// currency and asset not at zero has a row on From, carrying in what was held
// before it, and then one on each day its figure changes; a reader carries a
// figure forward until its next row, and an absent one is zero.
type NetWorthSeriesResult struct {
	Balances []NetWorthBalance `json:"balances"`
	Units    []NetWorthUnits   `json:"units"`
}
