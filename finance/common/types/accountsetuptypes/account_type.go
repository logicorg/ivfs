// Package accountsetuptypes holds the inputs and results of the account setup
// facade, which opens accounts and adds balances to them (ADR 0006).
package accountsetuptypes

import lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"

// AccountType says what a financial account holds, which decides the balances
// it is opened with (ADR 0006).
type AccountType string

// The account types.
const (
	// AccountTypeBank holds money: a current or savings account, a debit or
	// prepaid card, or a pension tracked only as a value.
	AccountTypeBank AccountType = "bank"
	// AccountTypeCard holds a debt: a credit or charge card.
	AccountTypeCard AccountType = "card"
	// AccountTypeBrokerage holds units of assets, and money: a broker, a crypto
	// exchange, an employer stock plan, or a pension invested in funds.
	AccountTypeBrokerage AccountType = "brokerage"
	// AccountTypeCash holds physical money, such as a wallet.
	AccountTypeCash AccountType = "cash"
)

// AccountTypes are the valid account types.
var AccountTypes = []AccountType{AccountTypeBank, AccountTypeCard, AccountTypeBrokerage, AccountTypeCash}

// PhysicalCashInstitution is the institution every cash account belongs to.
// It is created on first use, without asking, since its name is a constant
// and cannot be misspelt.
const PhysicalCashInstitution = "Physical cash"

// Outcome says what an account setup request did. Every outcome other than
// OutcomeOpened asks the caller a question and writes nothing.
type Outcome string

// The outcomes.
const (
	// OutcomeOpened means the account exists with its balances, created now
	// or before.
	OutcomeOpened Outcome = "opened"
	// OutcomeInstitutionNotFound means the institution does not exist and the
	// request did not ask to create it.
	OutcomeInstitutionNotFound Outcome = "institution_not_found"
)

// OpeningEntry is the journal entry that holds a balance's opening balance.
type OpeningEntry struct {
	LedgerAccountID int64 `json:"ledgerAccountId"`
	JournalEntryID  int64 `json:"journalEntryId"`
}

// Balances is what an opened account looks like afterwards: its balances and
// the shared accounts of their currencies, in the form ListAccounts returns,
// and the entries of the opening balances that were given.
type Balances struct {
	Balances       []lt.AccountSummary `json:"balances"`
	OpeningEntries []OpeningEntry      `json:"openingEntries,omitempty"`
}
