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

// LedgerAccountBalance returns a balance in the account's natural direction.
// For debit-normal accounts this is debits minus credits; for credit-normal
// accounts it is credits minus debits.
func (a *Activities) LedgerAccountBalance(ctx context.Context, params lt.LedgerAccountBalanceParams) (*lt.LedgerAccountBalanceResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.LedgerAccountID <= 0 {
		return nil, internalerrors.ErrInvalidParameter{Description: "ledger account ID must be positive"}
	}

	account := &model.LedgerAccount{}
	if err := model.FindLedgerAccountByIDQuery(db, params.LedgerAccountID).Relation("Currency").Scan(ctx, account); err != nil {
		if err == sql.ErrNoRows {
			return nil, internalerrors.ErrNotFound{ResourceType: "LedgerAccount", ID: params.LedgerAccountID}
		}
		return nil, err
	}

	var balance decimal.Decimal
	queryText := `
		SELECT COALESCE(SUM(CASE WHEN p.side = 'debit' THEN p.amount ELSE -p.amount END), 0)
		FROM ledger.posting AS p
		JOIN ledger.journal_entry AS je ON je.id = p.journal_entry_id
		WHERE p.ledger_account_id = ?
		  AND je.status = 'posted'`
	queryArgs := []any{params.LedgerAccountID}
	if !params.At.IsZero() {
		queryText += " AND je.occurred_at <= ?"
		queryArgs = append(queryArgs, params.At)
	}
	query := db.NewRaw(queryText, queryArgs...)
	if err := query.Scan(ctx, &balance); err != nil {
		return nil, fmt.Errorf("calculate ledger account balance: %w", err)
	}
	if account.Kind == model.LedgerAccountLiability || account.Kind == model.LedgerAccountEquity || account.Kind == model.LedgerAccountIncome {
		balance = balance.Neg()
	}
	return &lt.LedgerAccountBalanceResult{
		LedgerAccountID:   account.ID,
		LedgerAccountName: account.Name,
		Kind:              account.Kind,
		CurrencyID:        account.CurrencyID,
		CurrencyCode:      account.Currency.Code,
		Balance:           balance,
		Closed:            account.ClosedAt != nil,
	}, nil
}
