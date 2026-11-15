package ledgertypes

import (
	"strings"

	"github.com/logicorg/ivfs/pkg/model"
)

// OpeningBalancesAccountName is the name of the shared equity account, one per
// currency and of no financial account, that opening balances are booked
// against (ADR 0006).
const OpeningBalancesAccountName = "Opening balances"

// ExpensesAccountName and IncomeAccountName are the names of the shared
// accounts, one per currency and of no financial account, that every expense
// and income is booked on. What the money was for is the category of the
// posting, not an account.
const (
	ExpensesAccountName = "Expenses"
	IncomeAccountName   = "Income"
)

// UntrackedAccountsName is the name of the stray equity account, one per
// currency, that receives the postings of a deleted ledger account, and of
// the stray financial account that receives a deleted financial
// account's units (ADR 0012).
const UntrackedAccountsName = "Untracked accounts"

// UntrackedInstitutionName is the name of the institution that holds the
// stray financial account, "Untracked accounts" (ADR 0012).
const UntrackedInstitutionName = "Untracked"

// reservedLedgerAccounts are the shared and stray ledger accounts, by name,
// with the kind each has. Among the accounts of no financial account the name
// is theirs whatever the kind, so that a name says which account it is.
var reservedLedgerAccounts = map[string]model.LedgerAccountKind{
	strings.ToLower(OpeningBalancesAccountName): model.LedgerAccountEquity,
	strings.ToLower(FXClearingAccountName):      model.LedgerAccountAsset,
	strings.ToLower(ExpensesAccountName):        model.LedgerAccountExpense,
	strings.ToLower(IncomeAccountName):          model.LedgerAccountIncome,
	strings.ToLower(UntrackedAccountsName):      model.LedgerAccountEquity,
}

// ReservedLedgerAccountKind returns the kind of the shared or stray account
// of this name, and whether there is one. Names compare as the unique keys
// do, trimmed and case-insensitively (ADR 0005), so "fx clearing" is the one
// creation finds.
func ReservedLedgerAccountKind(name string) (model.LedgerAccountKind, bool) {
	kind, reserved := reservedLedgerAccounts[normalizeName(name)]
	return kind, reserved
}

// IsReservedLedgerAccount reports whether the account is one of the shared or
// stray accounts: it has no financial account and one of their names.
func IsReservedLedgerAccount(account *model.LedgerAccount) bool {
	if account.FinancialAccountID != nil {
		return false
	}
	_, reserved := ReservedLedgerAccountKind(account.Name)
	return reserved
}

// CategoryAccountName returns the name of the shared account that the
// postings of a category of this kind are booked on.
func CategoryAccountName(kind model.LedgerAccountKind) string {
	if kind == model.LedgerAccountIncome {
		return IncomeAccountName
	}
	return ExpensesAccountName
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
