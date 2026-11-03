package conversiontypes

import (
	"fmt"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// RetrieveRateHistoryParams asks for the rates from FromCurrency to
// ToCurrency on the days from From to To, both included. Only the dates of
// From and To count, not their times.
type RetrieveRateHistoryParams struct {
	FromCurrency int64
	ToCurrency   int64
	From         time.Time
	To           time.Time
}

// Validate checks both currencies are set and differ, and the range does not
// end before it starts.
func (p *RetrieveRateHistoryParams) Validate() error {
	var missing []string
	if p.FromCurrency == 0 {
		missing = append(missing, "FromCurrency")
	}
	if p.ToCurrency == 0 {
		missing = append(missing, "ToCurrency")
	}
	if p.From.IsZero() {
		missing = append(missing, "From")
	}
	if p.To.IsZero() {
		missing = append(missing, "To")
	}
	if len(missing) > 0 {
		return errors.ErrMissingRequiredParameter{Parameters: missing}
	}
	if p.FromCurrency == p.ToCurrency {
		return errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the rate from currency %d to itself is 1; ask for two different currencies", p.FromCurrency,
		)}
	}
	if p.To.Before(p.From) {
		return errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the history ends on %s, before it starts on %s",
			p.To.Format(time.DateOnly), p.From.Format(time.DateOnly),
		)}
	}
	return nil
}

// RetrieveRateHistoryResult holds the latest rate of each day that has one,
// oldest first. The first is the latest before From, when there is one, so a
// series over the range starts from what was known when the range began.
type RetrieveRateHistoryResult struct {
	Conversions []*model.Conversion
}
