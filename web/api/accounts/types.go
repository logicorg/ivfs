package accounts

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// FinancialAccount is an account held at an institution, as the API returns
// it.
type FinancialAccount struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	InstitutionID   int64  `json:"institutionId"`
	InstitutionName string `json:"institutionName"`
	// InstitutionClosed is whether the institution is closed, which it can
	// be only once each of its accounts is.
	InstitutionClosed bool `json:"institutionClosed"`
	Closed            bool `json:"closed"`
}

// FinancialAccountList is every financial account asked for, in no
// particular order.
type FinancialAccountList struct {
	FinancialAccounts []FinancialAccount `json:"financialAccounts"`
}

// ListFinancialAccountsInput is the request of GET /api/financial-accounts.
type ListFinancialAccountsInput struct {
	IncludeClosed bool `query:"includeClosed" doc:"also list closed accounts"`
}

// ListFinancialAccountsOutput is the response of GET /api/financial-accounts.
type ListFinancialAccountsOutput struct {
	Body FinancialAccountList
}

// Balance is the balance of one of a financial account's ledger accounts,
// such as its Cash in EUR, in the ledger account's natural direction: what
// an asset holds, or what a liability owes.
type Balance struct {
	LedgerAccountID   int64          `json:"ledgerAccountId"`
	LedgerAccountName string         `json:"ledgerAccountName"`
	Kind              string         `json:"kind" enum:"asset,liability,equity,income,expense"`
	CurrencyID        int64          `json:"currencyId"`
	CurrencyCode      string         `json:"currencyCode"`
	Balance           gateway.Amount `json:"balance"`
	Closed            bool           `json:"closed"`
}

// AccountBalances is every balance of a financial account.
type AccountBalances struct {
	FinancialAccountID int64     `json:"financialAccountId"`
	Balances           []Balance `json:"balances"`
}

// AccountBalancesInput is the request of GET
// /api/financial-accounts/{id}/balances.
type AccountBalancesInput struct {
	ID int64     `path:"id" minimum:"1"`
	At time.Time `query:"at" doc:"count only entries that occurred at or before this instant; now if omitted"`
}

// AccountBalancesOutput is the response of GET
// /api/financial-accounts/{id}/balances.
type AccountBalancesOutput struct {
	Body AccountBalances
}

// LedgerAccountBalanceInput is the request of GET
// /api/ledger-accounts/{id}/balance.
type LedgerAccountBalanceInput struct {
	ID int64     `path:"id" minimum:"1"`
	At time.Time `query:"at" doc:"count only entries that occurred at or before this instant; now if omitted"`
}

// LedgerAccountBalanceOutput is the response of GET
// /api/ledger-accounts/{id}/balance.
type LedgerAccountBalanceOutput struct {
	Body Balance
}

// newBalance is balance as the API returns it.
func newBalance(balance lt.LedgerAccountBalanceResult) Balance {
	return Balance{
		LedgerAccountID: balance.LedgerAccountID, LedgerAccountName: balance.LedgerAccountName,
		Kind: string(balance.Kind), CurrencyID: balance.CurrencyID, CurrencyCode: balance.CurrencyCode,
		Balance: gateway.NewAmount(balance.Balance), Closed: balance.Closed,
	}
}

// CloseInput is the request of closing an institution, a financial account,
// or a ledger account.
type CloseInput struct {
	ID int64 `path:"id" minimum:"1"`
}

// OpeningCurrency is one currency an account is opened in, with its opening
// balance.
type OpeningCurrency struct {
	Code           string         `json:"code" minLength:"1" doc:"ISO 4217 code, such as EUR"`
	OpeningBalance gateway.Amount `json:"openingBalance" required:"false" doc:"the balance today: money held, or owed on a card; negative for an overdraft; nothing is posted if omitted"`
}

// AccountForm is the answers to the form that opens an account (ADR 0006).
type AccountForm struct {
	Type              string            `json:"type" enum:"bank,card,brokerage,cash" doc:"bank holds money, card a credit card's debt, brokerage assets and money, cash a wallet"`
	InstitutionName   string            `json:"institutionName,omitempty" doc:"the institution that holds the account, such as Revolut; left out for a cash account"`
	CreateInstitution bool              `json:"createInstitution,omitempty" doc:"create the institution if it does not exist, once the person has confirmed its name"`
	AccountName       string            `json:"accountName,omitempty" doc:"the account's name; the institution's name, or Wallet for cash, if omitted"`
	Currencies        []OpeningCurrency `json:"currencies" minItems:"1"`
	OpeningBalancesAt time.Time         `json:"openingBalancesAt" required:"false" doc:"the instant the opening balances hold at; now if omitted"`
}

// OpenAccountInput is the request of POST /api/accounts/open.
type OpenAccountInput struct {
	Body AccountForm
}

// OpenedAccount is what opening an account did. For institution_not_found
// only the outcome and the institution's name are set, and nothing was
// written.
type OpenedAccount struct {
	Outcome              string `json:"outcome" enum:"opened,institution_not_found"`
	InstitutionID        int64  `json:"institutionId,omitempty"`
	InstitutionName      string `json:"institutionName"`
	FinancialAccountID   int64  `json:"financialAccountId,omitempty"`
	FinancialAccountName string `json:"financialAccountName,omitempty"`
}

// OpenAccountOutput is the response of POST /api/accounts/open.
type OpenAccountOutput struct {
	Body OpenedAccount
}

// BalanceForm is the answers to the form that adds a named balance, such as a
// savings vault, to a financial account (ADR 0006).
type BalanceForm struct {
	Name             string         `json:"name" minLength:"1" doc:"the balance's name, such as Savings vault"`
	CurrencyCode     string         `json:"currencyCode" minLength:"1" doc:"ISO 4217 code, such as EUR"`
	OpeningBalance   gateway.Amount `json:"openingBalance" required:"false" doc:"the money in it today; nothing is posted if omitted"`
	OpeningBalanceAt time.Time      `json:"openingBalanceAt" required:"false" doc:"the instant the opening balance holds at; now if omitted"`
}

// AddBalanceInput is the request of POST /api/financial-accounts/{id}/balances.
type AddBalanceInput struct {
	ID   int64 `path:"id" minimum:"1"`
	Body BalanceForm
}

// AddedBalance is the ledger account that holds the balance, created now or
// before.
type AddedBalance struct {
	LedgerAccountID   int64  `json:"ledgerAccountId"`
	LedgerAccountName string `json:"ledgerAccountName"`
}

// AddBalanceOutput is the response of POST
// /api/financial-accounts/{id}/balances.
type AddBalanceOutput struct {
	Body AddedBalance
}

// AccountName is a financial or ledger account's new name.
type AccountName struct {
	Name string `json:"name" minLength:"1"`
}

// RenameInput is the request of renaming a financial account or a ledger
// account.
type RenameInput struct {
	ID   int64 `path:"id" minimum:"1"`
	Body AccountName
}

// DeleteAccountInput is the request of deleting a ledger or a financial
// account.
type DeleteAccountInput struct {
	ID            int64 `path:"id" minimum:"1"`
	ReplacementID int64 `query:"replacementId" minimum:"1" doc:"the account to merge this one into; merging is not supported yet"`
}

// DeletionPreviewInput is the request of GET .../{id}/deletion-preview.
type DeletionPreviewInput struct {
	ID int64 `path:"id" minimum:"1"`
}

// DeleteInstitutionInput is the request of DELETE /api/institutions/{id}.
type DeleteInstitutionInput struct {
	ID int64 `path:"id" minimum:"1"`
}
