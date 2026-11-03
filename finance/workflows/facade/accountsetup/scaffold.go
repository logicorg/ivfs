// Package accountsetup is the facade that opens accounts and adds balances to
// them, composing the institution, account, and journal workflows (ADR 0006).
package accountsetup

import (
	"fmt"

	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// The names ADR 0006 gives the balances and shared accounts it creates.
const (
	cashBalance        = model.BrokerageCashBalance
	cardBalance        = "Card balance"
	investmentsBalance = model.BrokerageInvestmentsBalance
	openingBalances    = lt.OpeningBalancesAccountName
	fxClearing         = lt.FXClearingAccountName
	wallet             = "Wallet"
)

// balanceSpec is a ledger account the facade creates by name and kind, in a
// currency it is given.
type balanceSpec struct {
	name string
	kind model.LedgerAccountKind
}

// scaffold returns the balances an account of the given type gets in each of
// its currencies, the first being the one its opening balance goes to.
func scaffold(accountType ast.AccountType) []balanceSpec {
	switch accountType {
	case ast.AccountTypeCard:
		return []balanceSpec{{cardBalance, model.LedgerAccountLiability}}
	case ast.AccountTypeBrokerage:
		return []balanceSpec{{cashBalance, model.LedgerAccountAsset}, {investmentsBalance, model.LedgerAccountAsset}}
	default:
		return []balanceSpec{{cashBalance, model.LedgerAccountAsset}}
	}
}

// sharedAccounts returns the top-level accounts every currency gets once,
// shared by every account in it; "Opening balances" is the first.
func sharedAccounts() []balanceSpec {
	return []balanceSpec{{openingBalances, model.LedgerAccountEquity}, {fxClearing, model.LedgerAccountAsset}}
}

// institutionKind is the kind a new institution gets, from the type of the
// account it is created for.
func institutionKind(accountType ast.AccountType) string {
	switch accountType {
	case ast.AccountTypeCard:
		return "card issuer"
	case ast.AccountTypeBrokerage:
		return "broker"
	case ast.AccountTypeCash:
		return "cash"
	default:
		return "bank"
	}
}

// institutionName is the institution the account belongs to: the one the
// request names, or "Physical cash" for a cash account.
func institutionName(input ast.OpenAccountInput) string {
	if input.Type == ast.AccountTypeCash {
		return ast.PhysicalCashInstitution
	}
	return input.InstitutionName
}

// accountName is the financial account's name: the one the request gives, or
// else "Wallet" for a cash account and the institution's name for the others.
func accountName(input ast.OpenAccountInput) string {
	switch {
	case input.AccountName != "":
		return input.AccountName
	case input.Type == ast.AccountTypeCash:
		return wallet
	default:
		return input.InstitutionName
	}
}

// openingBalanceKey is the idempotency key of a balance's opening entry, so a
// balance has at most one however often it is opened.
func openingBalanceKey(ledgerAccountID int64) string {
	return fmt.Sprintf("opening-balance:%d", ledgerAccountID)
}

// openingBalancePostings books amount on a balance against "Opening balances".
// The amount is in the balance's natural direction, money held for an asset
// and money owed for a liability, so a negative one swaps the sides. Zero
// books nothing, and nil is returned.
func openingBalancePostings(balance int64, kind model.LedgerAccountKind, opening int64, amount decimal.Decimal) []lt.PostingInput {
	if amount.IsZero() {
		return nil
	}

	balanceSide, openingSide := model.PostingDebit, model.PostingCredit
	if kind == model.LedgerAccountLiability {
		balanceSide, openingSide = openingSide, balanceSide
	}
	if amount.IsNegative() {
		balanceSide, openingSide = openingSide, balanceSide
	}
	return []lt.PostingInput{
		{LedgerAccountID: balance, Amount: amount.Abs(), Side: balanceSide},
		{LedgerAccountID: opening, Amount: amount.Abs(), Side: openingSide},
	}
}
