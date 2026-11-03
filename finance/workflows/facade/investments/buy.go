package investments

import (
	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
)

// buildBuy is units bought with a brokerage account's cash: "Cash" is credited
// and "Investments" debited with the amount and the fee, and the units are
// added at that cost, since a fee is part of what they cost.
func buildBuy(in *it.BuyAssetInput, b *brokerage) trade {
	cost := in.Amount.Add(in.Fee)
	return trade{
		kind:               string(it.KindBuy),
		idempotencyKey:     in.IdempotencyKey,
		financialAccountID: in.FinancialAccountID,
		assetID:            in.AssetID,
		from:               b.cash,
		to:                 b.investments,
		amount:             cost,
		unitsDelta:         in.Units,
		costBasis:          &cost,
		currencyID:         b.currency.ID,
		occurredAt:         in.OccurredAt,
		description:        in.Description,
	}
}
