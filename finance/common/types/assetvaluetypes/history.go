package assetvaluetypes

import (
	"fmt"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// RetrieveAssetValueHistoryParams asks for the values of an asset observed
// from From to To, both included.
type RetrieveAssetValueHistoryParams struct {
	AssetID int64
	From    time.Time
	To      time.Time
}

// Validate checks the asset is set and the range does not end before it starts.
func (p *RetrieveAssetValueHistoryParams) Validate() error {
	var missing []string
	if p.AssetID == 0 {
		missing = append(missing, "AssetID")
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
	if p.To.Before(p.From) {
		return errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the history ends at %s, before it starts at %s",
			p.To.Format(time.RFC3339), p.From.Format(time.RFC3339),
		)}
	}
	return nil
}

// RetrieveAssetValueHistoryResult holds the values, oldest first. The first is
// the latest observed at or before From, when there is one, so a series over
// the range starts from what was known when the range began.
type RetrieveAssetValueHistoryResult struct {
	AssetValues []*model.AssetValue
}
