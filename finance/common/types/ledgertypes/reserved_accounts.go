package ledgertypes

import (
	"strings"

	"github.com/logicorg/ivfs/pkg/model"
)

//TODO: should we simplify, be stricter, and just reserve those names (regardless of kind)?
// we can get rid of the map and just use a simple array + slices.Contains()

// OpeningBalancesAccountName is the name of the shared equity account, one per
// currency and of no financial account, that opening balances are booked
// against (ADR 0006).
const OpeningBalancesAccountName = "Opening balances"

// UntrackedAccountsName is the name of the stray equity account, one per
// currency, that receives the postings of a deleted balance or equity account,
// and of the stray financial account that receives a deleted financial
// account's units (ADR 0012).
const UntrackedAccountsName = "Untracked accounts"

// UntrackedInstitutionName is the name of the institution that holds the
// stray financial account, "Untracked accounts" (ADR 0012).
const UntrackedInstitutionName = "Untracked"

// UncategorizedAccountName is the name of the stray category, one per kind and
// currency, that receives the postings of a deleted category (ADR 0012).
const UncategorizedAccountName = "Uncategorized"

// reservedLedgerAccounts are the kinds each shared or stray ledger account's
// name is reserved for.
var reservedLedgerAccounts = map[string][]model.LedgerAccountKind{
	strings.ToLower(OpeningBalancesAccountName): {model.LedgerAccountEquity},
	strings.ToLower(FXClearingAccountName):      {model.LedgerAccountAsset},
	strings.ToLower(UntrackedAccountsName):      {model.LedgerAccountEquity},
	strings.ToLower(UncategorizedAccountName):   {model.LedgerAccountExpense, model.LedgerAccountIncome},
}

// IsReservedLedgerAccount reports whether the account is one of the shared or
// stray accounts, which the ledger finds by name, kind, and being top-level
// with no financial account. Names compare as the unique keys do, trimmed and
// case-insensitively (ADR 0005), so "fx clearing" is the one creation finds.
func IsReservedLedgerAccount(account *model.LedgerAccount) bool {
	if account.ParentID != nil || account.FinancialAccountID != nil {
		return false
	}
	for _, kind := range reservedLedgerAccounts[normalizeName(account.Name)] {
		if account.Kind == kind {
			return true
		}
	}
	return false
}

// IsUntrackedInstitution reports whether an institution of this name is the
// one that holds the stray financial account. Institution names are unique,
// so the name alone identifies it.
func IsUntrackedInstitution(name string) bool {
	return normalizeName(name) == normalizeName(UntrackedInstitutionName)
}

// IsUntrackedFinancialAccountName reports whether a financial account of this
// name would be the stray financial account if it were under the "Untracked"
// institution, which the caller checks separately.
func IsUntrackedFinancialAccountName(name string) bool {
	return normalizeName(name) == normalizeName(UntrackedAccountsName)
}

// normalizeName compares names as the unique keys do (ADR 0005).
func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
