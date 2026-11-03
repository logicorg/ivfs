package portfolio

import (
	"github.com/logicorg/ivfs/mcp/tools/journal"
	"github.com/shopspring/decimal"
)

// RegisterInvestmentTradeToolInput mirrors RegisterInvestmentTradeInput with
// its embedded entry's IdempotencyKey dropped, for the same reason as
// journal.PostJournalEntryToolInput, whose shape it reuses directly.
type RegisterInvestmentTradeToolInput struct {
	Entry               journal.PostJournalEntryToolInput `json:"entry"`
	FinancialAccountID  int64                             `json:"financialAccountID"`
	AssetID             int64                             `json:"assetID"`
	UnitsDelta          decimal.Decimal                   `json:"unitsDelta"`
	CostBasisAmount     *decimal.Decimal                  `json:"costBasisAmount,omitempty"`
	CostBasisCurrencyID *int64                            `json:"costBasisCurrencyID,omitempty"`
}

// RegisterVestToolInput mirrors RegisterVestInput with its embedded entry's
// IdempotencyKey dropped, for the same reason.
type RegisterVestToolInput struct {
	Entry               journal.PostJournalEntryToolInput `json:"entry"`
	FinancialAccountID  int64                             `json:"financialAccountID"`
	AssetID             int64                             `json:"assetID"`
	Units               decimal.Decimal                   `json:"units"`
	CostBasisAmount     decimal.Decimal                   `json:"costBasisAmount"`
	CostBasisCurrencyID int64                             `json:"costBasisCurrencyID"`
}
