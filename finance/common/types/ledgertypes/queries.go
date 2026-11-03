package ledgertypes

import (
	"time"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

type ListPostingsParams struct {
	LedgerAccountID int64     `json:"ledgerAccountId"`
	At              time.Time `json:"at,omitempty"`
}

type ListPostingsResult struct {
	Postings []model.Posting `json:"postings"`
}

type TrialBalanceAtParams struct {
	At time.Time `json:"at,omitempty"`
}

type TrialBalanceRow struct {
	LedgerAccountID int64           `json:"ledgerAccountId"`
	CurrencyID      int64           `json:"currencyId"`
	Debit           decimal.Decimal `json:"debit"`
	Credit          decimal.Decimal `json:"credit"`
	Balance         decimal.Decimal `json:"balance"`
}

type TrialBalanceAtResult struct {
	Rows []TrialBalanceRow `json:"rows"`
}
