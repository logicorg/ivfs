package reporting

import (
	"context"
	"database/sql"
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// row is a TrialBalanceAt row.
type row struct {
	LedgerAccountID int64           `bun:"ledger_account_id"`
	CurrencyID      int64           `bun:"currency_id"`
	Debit           decimal.Decimal `bun:"debit"`
	Credit          decimal.Decimal `bun:"credit"`
	Balance         decimal.Decimal `bun:"balance"`
}

// ToTrialBalanceRow converts a row to a ledgertypes `TrialBalanceRow`.
func (r *row) ToTrialBalanceRow() lt.TrialBalanceRow {
	return lt.TrialBalanceRow{
		LedgerAccountID: r.LedgerAccountID,
		CurrencyID:      r.CurrencyID,
		Debit:           r.Debit,
		Credit:          r.Credit,
		Balance:         r.Balance,
	}
}

// ListPostings lists postings of a LedgerAccount
func (a *Activities) ListPostings(ctx context.Context, params lt.ListPostingsParams) (*lt.ListPostingsResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.LedgerAccountID <= 0 {
		return nil, internalerrors.ErrInvalidParameter{Description: "ledger account ID must be positive"}
	}
	var account model.LedgerAccount
	if err := db.NewSelect().Model(&account).Where("id = ?", params.LedgerAccountID).Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return nil, internalerrors.ErrNotFound{ResourceType: "LedgerAccount", ID: params.LedgerAccountID}
		}
		return nil, fmt.Errorf("load ledger account: %w", err)
	}
	query := model.FindPostingsByLedgerAccountIDQuery(db, params.LedgerAccountID).
		Relation("LedgerAccount").
		Join("JOIN ledger.journal_entry AS je ON je.id = p.journal_entry_id").
		Where("je.status = ?", model.JournalEntryPosted)
	if !params.At.IsZero() {
		query = query.Where("je.occurred_at <= ?", params.At)
	}
	var postings []model.Posting
	if err := query.Scan(ctx, &postings); err != nil {
		return nil, fmt.Errorf("list postings: %w", err)
	}
	return &lt.ListPostingsResult{Postings: postings}, nil
}

func (a *Activities) TrialBalanceAt(ctx context.Context, params lt.TrialBalanceAtParams) (*lt.TrialBalanceAtResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	query := db.NewSelect().
		TableExpr("ledger.ledger_account AS la").
		ColumnExpr("la.id AS ledger_account_id").
		ColumnExpr("la.currency_id AS currency_id").
		ColumnExpr("COALESCE(SUM(CASE WHEN p.side = 'debit' THEN p.amount ELSE 0 END), 0) AS debit").
		ColumnExpr("COALESCE(SUM(CASE WHEN p.side = 'credit' THEN p.amount ELSE 0 END), 0) AS credit").
		ColumnExpr("COALESCE(SUM(CASE WHEN p.side = 'debit' THEN p.amount ELSE -p.amount END), 0) AS balance").
		// A posting and its entry are joined together, and the conditions that
		// decide whether the posting counts sit in the ON, not in a WHERE: an
		// account whose postings are all drafts, or all after At, then keeps its
		// row with nothing joined to it, at zero, instead of being filtered out.
		Join("LEFT JOIN (ledger.posting AS p JOIN ledger.journal_entry AS je ON je.id = p.journal_entry_id)").
		JoinOn("p.ledger_account_id = la.id").
		JoinOn("je.status = ?", model.JournalEntryPosted)
	if !params.At.IsZero() {
		query = query.JoinOn("je.occurred_at <= ?", params.At)
	}
	query = query.Group("la.id", "la.currency_id").Order("la.id")

	var rows []row
	if err := query.Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("calculate trial balance: %w", err)
	}
	result := &lt.TrialBalanceAtResult{Rows: make([]lt.TrialBalanceRow, 0, len(rows))}
	for _, row := range rows {
		result.Rows = append(result.Rows, row.ToTrialBalanceRow())
	}
	return result, nil
}
