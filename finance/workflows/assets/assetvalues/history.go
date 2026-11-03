package assetvalues

import (
	"fmt"
	"time"

	"github.com/logicorg/ivfs/finance/activities/asset/value"
	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// RetrieveAssetValueHistory retrieves the values of an asset observed in a
// range, with the one carried into it. It joins two existing activities
// rather than adding one with an upper bound: values entered by hand are few,
// so reading past To and dropping the rest costs little.
func RetrieveAssetValueHistory(ctx workflow.Context, params avt.RetrieveAssetValueHistoryParams) (*avt.RetrieveAssetValueHistoryResult, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var ava *value.AssetValueActivity
	carriedIn := workflow.ExecuteActivity(ctx, ava.RetrieveAssetValue, avt.RetrieveAssetValueParams{
		AssetID: params.AssetID, Date: &params.From,
	})
	observed := workflow.ExecuteActivity(ctx, ava.RetrieveAssetValues, avt.RetrieveAssetValuesParams{
		AssetID: params.AssetID, Since: &params.From,
	})

	values := make([]*model.AssetValue, 0)
	var latest avt.RetrieveAssetValueResult
	err := carriedIn.Get(ctx, &latest)
	switch {
	case err == nil:
		values = append(values, latest.AssetValue)
	case workflows.HasErrorType(err, "ErrNotFound"):
		// Nothing was observed before the range: the series starts later.
	default:
		return nil, fmt.Errorf("retrieve the asset value at %s: %w", params.From.Format(time.RFC3339), err)
	}

	var since avt.RetrieveAssetValuesResult
	if err := observed.Get(ctx, &since); err != nil {
		return nil, fmt.Errorf("retrieve the asset values since %s: %w", params.From.Format(time.RFC3339), err)
	}
	for _, v := range since.AssetValues {
		if v.ObservedAt.After(params.To) {
			break
		}
		// A value observed exactly at From is also the one carried in.
		if latest.AssetValue != nil && v.ID == latest.AssetValue.ID {
			continue
		}
		values = append(values, v)
	}
	return &avt.RetrieveAssetValueHistoryResult{AssetValues: values}, nil
}
