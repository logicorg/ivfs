package investments

import (
	"fmt"

	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// UpdateInvestment makes a journal entry an investment of the given kind,
// whatever it was before, with the lookups and the builders of
// RecordInvestment. The entry keeps its id, and its movement is replaced by
// the one the builder makes. The holdings left negative, of the asset the
// entry moved before and of the one it moves now, are returned as warnings:
// unlike recording one, updating a vest can warn, when it lowers units that
// have since been sold.
func UpdateInvestment(ctx workflow.Context, input it.UpdateInvestmentInput) (*it.TradeResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	switch input.Kind {
	case it.KindBuy:
		buy := input.Buy.Buy()
		b, err := retrieveBrokerage(ctx, buy.FinancialAccountID, buy.CurrencyCode)
		if err != nil {
			return nil, err
		}
		return replaceTrade(ctx, input.JournalEntryID, input.Kind, buildBuy(buy, b))
	case it.KindSell:
		sell := input.Sell.Sell()
		b, err := retrieveBrokerage(ctx, sell.FinancialAccountID, sell.CurrencyCode)
		if err != nil {
			return nil, err
		}
		return replaceTrade(ctx, input.JournalEntryID, input.Kind, buildSell(sell, b))
	case it.KindVest:
		vest := input.Vest.Vest()
		b, err := retrieveBrokerage(ctx, vest.FinancialAccountID, vest.CurrencyCode)
		if err != nil {
			return nil, err
		}
		categories, err := retrieveVestCategories(ctx, vest, b.currency)
		if err != nil {
			return nil, err
		}
		built := buildVest(vest, b, categories)
		return replace(ctx, input.JournalEntryID, input.Kind, built.Entry, lt.ReplaceMovementInput{
			Kind:                model.PositionMovementVest,
			FinancialAccountID:  built.FinancialAccountID,
			AssetID:             built.AssetID,
			UnitsDelta:          built.Units,
			CostBasisAmount:     &built.CostBasisAmount,
			CostBasisCurrencyID: &built.CostBasisCurrencyID,
		})
	}
	// Validate refuses any other kind, so this is a kind added there and not
	// here.
	return nil, internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("kind %q cannot be updated to yet", input.Kind)}
}

// replaceTrade replaces the entry with the trade's entry and movement.
func replaceTrade(ctx workflow.Context, journalEntryID int64, kind it.Kind, t trade) (*it.TradeResult, error) {
	built := t.input()
	return replace(ctx, journalEntryID, kind, built.Entry, lt.ReplaceMovementInput{
		Kind:                model.PositionMovementTrade,
		FinancialAccountID:  built.FinancialAccountID,
		AssetID:             built.AssetID,
		UnitsDelta:          built.UnitsDelta,
		CostBasisAmount:     built.CostBasisAmount,
		CostBasisCurrencyID: built.CostBasisCurrencyID,
	})
}
