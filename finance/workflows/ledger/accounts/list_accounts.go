package accounts

import (
	"fmt"
	"strings"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// ListAccounts lists ledger accounts together with the names of their
// financial account and institution, so a caller can resolve something like
// "revolut eur" to a ledger account id without knowing how the chart of
// accounts is built. It joins the results of ListLedgerAccounts and
// ListFinancialAccounts rather than reading through an activity of its own:
// the two reads are not one snapshot, which only matters for names changed
// between them.
func ListAccounts(ctx workflow.Context, params lt.ListAccountsParams) (*lt.ListAccountsResult, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	// The ledger accounts are read first: a financial account cannot be
	// deleted while a ledger account refers to it, so every one they refer to
	// is still there for the second read.
	var ledgerAccounts lt.ListLedgerAccountsResult
	if err := workflow.ExecuteActivity(ctx, la.ListLedgerAccounts, lt.ListLedgerAccountsParams{
		Kinds: params.Kinds, IncludeClosed: params.IncludeClosed,
	}).Get(ctx, &ledgerAccounts); err != nil {
		return nil, fmt.Errorf("list ledger accounts: %w", err)
	}

	var financialAccounts lt.ListFinancialAccountsResult
	if err := workflow.ExecuteActivity(ctx, la.ListFinancialAccounts, lt.ListFinancialAccountsParams{
		IncludeClosed: true,
	}).Get(ctx, &financialAccounts); err != nil {
		return nil, fmt.Errorf("list financial accounts: %w", err)
	}
	byID := make(map[int64]model.FinancialAccount, len(financialAccounts.FinancialAccounts))
	for _, account := range financialAccounts.FinancialAccounts {
		byID[account.ID] = account
	}

	accounts := make([]lt.AccountSummary, 0, len(ledgerAccounts.LedgerAccounts))
	for _, ledgerAccount := range ledgerAccounts.LedgerAccounts {
		summary := lt.AccountSummary{
			LedgerAccountID:    ledgerAccount.ID,
			LedgerAccountName:  ledgerAccount.Name,
			Kind:               ledgerAccount.Kind,
			ParentID:           ledgerAccount.ParentID,
			CurrencyID:         ledgerAccount.CurrencyID,
			FinancialAccountID: ledgerAccount.FinancialAccountID,
			Closed:             ledgerAccount.ClosedAt != nil,
		}
		if ledgerAccount.Currency == nil {
			// ListLedgerAccounts always loads the currency, which every ledger
			// account has, so this is a bug. The panic blocks the workflow task,
			// which Temporal retries until a fixed worker completes it.
			panic(fmt.Sprintf("ledger account %d has no currency loaded", ledgerAccount.ID))
		}
		summary.CurrencyCode = ledgerAccount.Currency.Code
		if id := ledgerAccount.FinancialAccountID; id != nil {
			// ListFinancialAccounts stops at MaxFinancialAccountsResult, so
			// past that a referenced account can be missing from it.
			financialAccount, ok := byID[*id]
			if !ok {
				return nil, fmt.Errorf("financial account %d of ledger account %d was not listed; at most %d are",
					*id, ledgerAccount.ID, lt.MaxFinancialAccountsResult)
			}
			summary.FinancialAccountName = financialAccount.Name
			if financialAccount.Institution != nil {
				summary.InstitutionName = financialAccount.Institution.Name
			}
			summary.Closed = summary.Closed || financialAccount.ClosedAt != nil
		}

		if summary.Closed && !params.IncludeClosed {
			continue
		}
		if matchesAccountFilters(summary, params) {
			accounts = append(accounts, summary)
		}
	}
	return &lt.ListAccountsResult{Accounts: accounts}, nil
}

// matchesAccountFilters applies the filters ListLedgerAccounts cannot, since
// they need the names it does not load.
func matchesAccountFilters(summary lt.AccountSummary, params lt.ListAccountsParams) bool {
	switch params.Scope {
	case lt.AccountScopeFinancial:
		if summary.FinancialAccountID == nil {
			return false
		}
	case lt.AccountScopeUnlinked:
		if summary.FinancialAccountID != nil {
			return false
		}
	}
	if params.CurrencyCode != "" && !strings.EqualFold(summary.CurrencyCode, params.CurrencyCode) {
		return false
	}

	names := strings.ToLower(strings.Join([]string{
		summary.LedgerAccountName, summary.FinancialAccountName, summary.InstitutionName, summary.CurrencyCode,
	}, " "))
	for _, word := range strings.Fields(strings.ToLower(params.Search)) {
		if !strings.Contains(names, word) {
			return false
		}
	}
	return true
}
