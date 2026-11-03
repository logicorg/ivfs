// Package investments records the investments a person thinks in, a buy, a
// sale, or a vest, as one journal entry and one position movement each,
// deciding the sides and the basis in code (ADR 0009).
package investments

import (
	"fmt"
	"strings"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	portfoliowf "github.com/logicorg/ivfs/finance/workflows/ledger/portfolio"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/workflow"
)

// The balances ADR 0006 gives a brokerage account in each of its currencies.
const (
	cashBalance        = model.BrokerageCashBalance
	investmentsBalance = model.BrokerageInvestmentsBalance
)

// brokerage is the "Cash" and "Investments" balances of a brokerage account in
// one currency.
type brokerage struct {
	currency    *model.Currency
	cash        int64
	investments int64
}

// trade is cash and units moving between the two balances of a brokerage,
// the rule a buy and a sale share.
type trade struct {
	// kind prefixes the idempotency key, so the same key sent for a buy and a
	// sale cannot return the other's entry.
	kind               string
	idempotencyKey     string
	financialAccountID int64
	assetID            int64
	// from and to are the ledger accounts the cash leaves and reaches.
	from, to   int64
	amount     decimal.Decimal
	unitsDelta decimal.Decimal
	// costBasis is nil for a sale, whose basis is worked out when read, and
	// currencyID is its currency, ignored without it.
	costBasis   *decimal.Decimal
	currencyID  int64
	occurredAt  time.Time
	description string
}

// retrieveBrokerage finds the "Cash" and "Investments" balances of the
// financial account in the currency, by the key a balance is unique by. A
// closed one is found too, so the trade fails on it with ErrResourceClosed
// rather than with a missing balance. An unknown currency fails with
// ErrNotFound in the error's cause chain.
func retrieveBrokerage(ctx workflow.Context, financialAccountID int64, currencyCode string) (*brokerage, error) {
	code := strings.ToUpper(strings.TrimSpace(currencyCode))
	var currency model.Currency
	err := workflow.ExecuteChildWorkflow(ctx, currencieswf.RetrieveCurrency, &ct.RetrieveCurrencyParams{Code: code}).Get(ctx, &currency)
	if err != nil {
		return nil, fmt.Errorf("retrieve currency %s: %w", code, err)
	}

	var listed lt.ListLedgerAccountsResult
	err = workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.ListLedgerAccounts, lt.ListLedgerAccountsParams{
		FinancialAccountID: &financialAccountID,
		CurrencyID:         &currency.ID,
		Kinds:              []model.LedgerAccountKind{model.LedgerAccountAsset},
		IncludeClosed:      true,
	}).Get(ctx, &listed)
	if err != nil {
		return nil, fmt.Errorf("list the balances of financial account %d: %w", financialAccountID, err)
	}

	result := &brokerage{currency: &currency}
	for _, account := range listed.LedgerAccounts {
		switch {
		case strings.EqualFold(account.Name, cashBalance):
			result.cash = account.ID
		case account.IsBrokerageInvestments():
			result.investments = account.ID
		}
	}
	if result.cash == 0 || result.investments == 0 {
		return nil, internalerrors.ErrInvalidParameter{
			Description: fmt.Sprintf(
				"financial account %d has no %q and %q balances in %s: open it again as a brokerage with %s to add them",
				financialAccountID, cashBalance, investmentsBalance, code, code,
			),
		}
	}
	return result, nil
}

// register writes the trade's entry and movement in one transaction. A closed
// account or an unknown asset fails here, inside it.
func register(ctx workflow.Context, t trade) (*it.TradeResult, error) {
	input := lt.RegisterInvestmentTradeInput{
		Entry: lt.PostJournalEntryInput{
			IdempotencyKey: t.kind + ":" + t.idempotencyKey,
			OccurredAt:     t.occurredAt,
			Postings: []lt.PostingInput{
				{LedgerAccountID: t.to, Amount: t.amount, Side: model.PostingDebit},
				{LedgerAccountID: t.from, Amount: t.amount, Side: model.PostingCredit},
			},
		},
		FinancialAccountID: t.financialAccountID,
		AssetID:            t.assetID,
		UnitsDelta:         t.unitsDelta,
	}
	if t.costBasis != nil {
		input.CostBasisAmount, input.CostBasisCurrencyID = t.costBasis, &t.currencyID
	}
	if t.description != "" {
		input.Entry.Description = &t.description
	}

	var result lt.RegisterInvestmentTradeResult
	if err := workflow.ExecuteChildWorkflow(ctx, portfoliowf.RegisterInvestmentTrade, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("register %s: %w", t.kind, err)
	}
	return &it.TradeResult{
		JournalEntryID:     result.JournalEntryID,
		PositionMovementID: result.PositionMovementID,
		Warnings:           result.Warnings,
	}, nil
}

// registerVest writes the vest's entry and movement in one transaction. A
// closed account or category, or an unknown asset, fails here, inside it.
func registerVest(ctx workflow.Context, input lt.RegisterVestInput) (*it.TradeResult, error) {
	var result lt.RegisterVestResult
	if err := workflow.ExecuteChildWorkflow(ctx, portfoliowf.RegisterVest, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("register vest: %w", err)
	}
	return &it.TradeResult{JournalEntryID: result.JournalEntryID, PositionMovementID: result.PositionMovementID}, nil
}

// retrieveLedgerAccounts retrieves the given ledger accounts concurrently.
func retrieveLedgerAccounts(ctx workflow.Context, ids ...int64) ([]*model.LedgerAccount, error) {
	futures := make([]workflow.ChildWorkflowFuture, len(ids))
	for i, id := range ids {
		futures[i] = workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{ID: id})
	}

	accounts := make([]*model.LedgerAccount, len(ids))
	for i, future := range futures {
		var result lt.RetrieveLedgerAccountResult
		if err := future.Get(ctx, &result); err != nil {
			return nil, fmt.Errorf("retrieve ledger account %d: %w", ids[i], err)
		}
		accounts[i] = result.LedgerAccount
	}
	return accounts, nil
}
