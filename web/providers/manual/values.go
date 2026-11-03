package manual

import (
	"context"
	"time"

	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/finance/workflows/assets/assetvalues"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// Values answers the values of asset entered by hand.
func (p *Provider) Values(ctx context.Context, asset model.Asset, r assetvalue.Range) ([]assetvalue.Value, error) {
	dates, err := r.Dates()
	if err != nil {
		return nil, err
	}
	history, err := gateway.RunWorkflow[avt.RetrieveAssetValueHistoryResult](ctx, p.c, assetvalues.RetrieveAssetValueHistory,
		avt.RetrieveAssetValueHistoryParams{
			AssetID: asset.ID,
			From:    dates[0],
			// The last date's value may be observed until its day ends.
			To: dates[len(dates)-1].AddDate(0, 0, 1).Add(-time.Nanosecond),
		})
	if err != nil {
		return nil, err
	}
	series := assetvalue.CarryForward(dates, history.AssetValues, func(v *model.AssetValue) time.Time { return v.ObservedAt })
	if len(series) == 0 {
		return nil, errors.ErrNotFound{ResourceType: "AssetValue", ID: asset.Name}
	}

	codes, err := p.currencyCodes(ctx)
	if err != nil {
		return nil, err
	}
	values := make([]assetvalue.Value, 0, len(series))
	for _, d := range series {
		code, ok := codes[d.Row.CurrencyID]
		if !ok {
			return nil, errors.ErrNotFound{ResourceType: "Currency", ID: d.Row.CurrencyID}
		}
		values = append(values, assetvalue.Value{
			Date: d.Date, Amount: d.Row.Value, Currency: code, ObservedAt: d.Row.ObservedAt,
		})
	}
	return values, nil
}
