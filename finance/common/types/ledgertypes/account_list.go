// Package ledgertypes holds the inputs and results of the ledger activities
// and workflows.
package ledgertypes

import (
	"fmt"

	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// AccountScope narrows ListAccounts to the ledger accounts that belong to a
// financial account, or to those that belong to none, such as categories and
// clearing accounts.
type AccountScope string

// The account scopes; AccountScopeAll, the zero value, lists every account.
const (
	AccountScopeAll       AccountScope = ""
	AccountScopeFinancial AccountScope = "financial"
	AccountScopeUnlinked  AccountScope = "unlinked"
)

// AccountScopes are the valid account scopes.
var AccountScopes = []AccountScope{AccountScopeAll, AccountScopeFinancial, AccountScopeUnlinked}

// ListAccountsParams searches every ledger account together with the
// financial account and institution it belongs to, the way a caller resolving
// something like "revolut eur" to a ledger account id would.
type ListAccountsParams struct {
	Search        string                    `json:"search,omitempty" jsonschema:"words that must each appear, case-insensitively, in the ledger account's, financial account's, or institution's name, or the currency code, such as \"revolut eur\""`
	CurrencyCode  string                    `json:"currencyCode,omitempty" jsonschema:"keep only accounts in this currency, such as EUR"`
	Kinds         []model.LedgerAccountKind `json:"kinds,omitempty" jsonschema:"keep only accounts of these kinds: asset, liability, equity, income, or expense"`
	Scope         AccountScope              `json:"scope,omitempty" jsonschema:"financial for accounts that belong to a financial account, unlinked for those that belong to none, such as categories; both if omitted"`
	IncludeClosed bool                      `json:"includeClosed,omitempty" jsonschema:"also list closed accounts, and accounts of a closed financial account"`
}

// Validate checks the scope, which CUE cannot express as an enum of a named
// string type; the kinds are validated by ListLedgerAccounts.
func (p *ListAccountsParams) Validate() error {
	switch p.Scope {
	case AccountScopeAll, AccountScopeFinancial, AccountScopeUnlinked:
		return nil
	default:
		err := fmt.Errorf("unknown account scope %q (must be one of %q)", p.Scope, AccountScopes)
		return errs.ValidationError{Err: err}
	}
}

// AccountSummary is one ledger account with the names a person knows it by.
// The financial account and institution fields are empty for a ledger account
// that belongs to no financial account.
type AccountSummary struct {
	LedgerAccountID      int64                   `json:"ledgerAccountId"`
	LedgerAccountName    string                  `json:"ledgerAccountName"`
	Kind                 model.LedgerAccountKind `json:"kind"`
	ParentID             *int64                  `json:"parentId,omitempty"`
	CurrencyID           int64                   `json:"currencyId"`
	CurrencyCode         string                  `json:"currencyCode"`
	FinancialAccountID   *int64                  `json:"financialAccountId,omitempty"`
	FinancialAccountName string                  `json:"financialAccountName,omitempty"`
	InstitutionName      string                  `json:"institutionName,omitempty"`
	// Closed is true when the ledger account or its financial account is closed.
	Closed bool `json:"closed,omitempty"`
}

// ListAccountsResult is every matching account, in ListLedgerAccounts' order.
type ListAccountsResult struct {
	Accounts []AccountSummary `json:"accounts"`
}
