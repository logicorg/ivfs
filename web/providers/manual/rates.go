package manual

import (
	"context"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/conversiontypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// Rates answers the rates from base to quote entered by hand. Only that
// direction is read: a rate entered from quote to base is not inverted.
func (p *Provider) Rates(ctx context.Context, base, quote string, r assetvalue.Range) ([]assetvalue.Rate, error) {
	dates, err := r.Dates()
	if err != nil {
		return nil, err
	}
	noRates := errors.ErrNotFound{ResourceType: "Conversion", ID: base + "/" + quote}
	codes, err := p.currencyCodes(ctx)
	if err != nil {
		return nil, err
	}
	var baseID, quoteID int64
	for id, code := range codes {
		switch code {
		case base:
			baseID = id
		case quote:
			quoteID = id
		}
	}
	// A currency the user does not have has no rates entered for it.
	if baseID == 0 || quoteID == 0 {
		return nil, noRates
	}

	history, err := gateway.RunWorkflow[ct.RetrieveRateHistoryResult](ctx, p.c, currencieswf.RetrieveConversionRateHistory,
		ct.RetrieveRateHistoryParams{FromCurrency: baseID, ToCurrency: quoteID, From: dates[0], To: dates[len(dates)-1]})
	if err != nil {
		return nil, err
	}
	series := assetvalue.CarryForward(dates, history.Conversions, func(c *model.Conversion) time.Time { return c.ObservedAt })
	if len(series) == 0 {
		return nil, noRates
	}
	rates := make([]assetvalue.Rate, 0, len(series))
	for _, d := range series {
		rates = append(rates, assetvalue.Rate{Date: d.Date, Rate: d.Row.Rate, ObservedAt: d.Row.ObservedAt})
	}
	return rates, nil
}
