package currencies

import (
	"fmt"
	"slices"
	"time"

	"github.com/logicorg/ivfs/finance/activities/currency/conversion"
	ct "github.com/logicorg/ivfs/finance/common/types/conversiontypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// ratePageSize bounds each read. A decade of daily rates is about 37 pages,
// far below the workflow history's limit.
const ratePageSize = 100

// RetrieveConversionRateHistory retrieves the rate of each day in a range,
// with the one carried into it. It pages the existing activity backwards from
// To, each page asking for the days before the oldest one it has, until it
// reaches a rate before From or runs out.
func RetrieveConversionRateHistory(ctx workflow.Context, params ct.RetrieveRateHistoryParams) (*ct.RetrieveRateHistoryResult, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var ca *conversion.Conversion
	from := dateOf(params.From)
	// Newest first while paging.
	rates := make([]*model.Conversion, 0)
	for date := dateOf(params.To); ; {
		var page ct.RetrieveResponse
		err := workflow.ExecuteActivity(ctx, ca.RetrieveMostRecentCurrencyConversionRatesAtDate, ct.RetrieveRequest{
			FromCurrency: params.FromCurrency, ToCurrency: params.ToCurrency, Date: date,
		}, int64(ratePageSize)).Get(ctx, &page)
		if workflows.HasErrorType(err, "ErrNotFound") {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("retrieve the rates on or before %s: %w", date.Format(time.DateOnly), err)
		}

		carriedIn := false
		for _, c := range page.Conversions {
			// A day's latest rate comes first; the rest of that day is older.
			if len(rates) > 0 && observedOn(rates[len(rates)-1]).Equal(observedOn(c)) {
				continue
			}
			rates = append(rates, c)
			if carriedIn = observedOn(c).Before(from); carriedIn {
				break
			}
		}
		if carriedIn || len(page.Conversions) < ratePageSize {
			break
		}
		// A day the page cut short has its latest rate already, so the next
		// page starts the day before.
		date = observedOn(page.Conversions[len(page.Conversions)-1]).AddDate(0, 0, -1)
	}

	slices.Reverse(rates)
	return &ct.RetrieveRateHistoryResult{Conversions: rates}, nil
}

// dateOf is t's own date, at midnight UTC.
func dateOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// observedOn is the day a rate was observed in UTC, the zone the activity's
// DATE(observed_at) dates it in on the database.
func observedOn(c *model.Conversion) time.Time {
	return dateOf(c.ObservedAt.UTC())
}
