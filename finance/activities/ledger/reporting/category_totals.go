package reporting

import (
	"context"
	"fmt"

	rpt "github.com/logicorg/ivfs/finance/common/types/reportingtypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// categoryTotal is a CategoryTotals row, with its total as debits less
// credits.
type categoryTotal struct {
	CategoryID   *int64                  `bun:"category_id"`
	Kind         model.LedgerAccountKind `bun:"kind"`
	CurrencyID   int64                   `bun:"currency_id"`
	CurrencyCode string                  `bun:"currency_code"`
	Debits       decimal.Decimal         `bun:"debits"`
}

// CategoryTotals totals the postings to income and expense ledger accounts
// over the posted entries of a period, by their category and the account's
// currency, in the account's natural direction. The postings without a
// category make one total per kind and currency.
func (a *Activities) CategoryTotals(ctx context.Context, params rpt.CategoryTotalsParams) (*rpt.CategoryTotalsResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	var rows []categoryTotal
	err = db.NewSelect().
		TableExpr("ledger.posting AS p").
		Join("JOIN ledger.journal_entry AS je ON je.id = p.journal_entry_id").
		Join("JOIN ledger.ledger_account AS la ON la.id = p.ledger_account_id").
		Join("JOIN currency.currency AS c ON c.id = la.currency_id").
		ColumnExpr("p.category_id").
		ColumnExpr("la.kind").
		ColumnExpr("la.currency_id").
		ColumnExpr("c.code AS currency_code").
		ColumnExpr("SUM(CASE WHEN p.side = 'debit' THEN p.amount ELSE -p.amount END) AS debits").
		Where("je.status = ?", model.JournalEntryPosted).
		Where("la.kind IN (?, ?)", model.LedgerAccountIncome, model.LedgerAccountExpense).
		Where("je.occurred_at >= ?", params.From).
		Where("je.occurred_at < ?", params.To).
		Group("p.category_id", "la.kind", "la.currency_id", "c.code").
		OrderExpr("p.category_id ASC NULLS LAST, la.kind, la.currency_id").
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("total categories: %w", err)
	}

	result := &rpt.CategoryTotalsResult{Totals: make([]rpt.CategoryTotal, 0, len(rows))}
	for _, row := range rows {
		total := row.Debits
		if row.Kind == model.LedgerAccountIncome {
			total = total.Neg()
		}
		result.Totals = append(result.Totals, rpt.CategoryTotal{
			CategoryID:   row.CategoryID,
			Kind:         row.Kind,
			CurrencyID:   row.CurrencyID,
			CurrencyCode: row.CurrencyCode,
			Total:        total,
		})
	}
	return result, nil
}
