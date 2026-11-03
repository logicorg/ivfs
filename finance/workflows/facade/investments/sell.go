package investments

import (
	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
)

// buildSell is units sold for a brokerage account's cash: "Investments" is
// credited and "Cash" debited with the amount less the fee, and the units are
// removed with no basis, so the average cost they remove, and the gain, are
// worked out when read and follow any later backdated movement.
func buildSell(in *it.SellAssetInput, b *brokerage) trade {
	return trade{
		kind:               string(it.KindSell),
		idempotencyKey:     in.IdempotencyKey,
		financialAccountID: in.FinancialAccountID,
		assetID:            in.AssetID,
		from:               b.investments,
		to:                 b.cash,
		amount:             in.Amount.Sub(in.Fee),
		unitsDelta:         in.Units.Neg(),
		occurredAt:         in.OccurredAt,
		description:        in.Description,
	}
}
