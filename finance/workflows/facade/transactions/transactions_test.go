package transactions

import (
	"maps"
	"slices"
	"testing"
	"time"

	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type WorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestWorkflowUnitTestSuite(t *testing.T) { suite.Run(t, new(WorkflowUnitTestSuite)) }

// The ledger accounts the transactions are recorded between, as
// RetrieveLedgerAccount reports them: the balances of Revolut Personal (10)
// and of a mortgage (11), and the shared accounts, which belong to no
// financial account.
var (
	cash           = &model.LedgerAccount{ID: 100, FinancialAccountID: ptr.Int64(10), CurrencyID: 1, Name: "Cash", Kind: model.LedgerAccountAsset}
	cardBalance    = &model.LedgerAccount{ID: 101, FinancialAccountID: ptr.Int64(10), CurrencyID: 1, Name: "Card balance", Kind: model.LedgerAccountLiability}
	savingsVault   = &model.LedgerAccount{ID: 102, FinancialAccountID: ptr.Int64(10), CurrencyID: 1, Name: "Savings vault", Kind: model.LedgerAccountAsset}
	cashUSD        = &model.LedgerAccount{ID: 103, FinancialAccountID: ptr.Int64(10), CurrencyID: 2, Name: "Cash (USD)", Kind: model.LedgerAccountAsset}
	principal      = &model.LedgerAccount{ID: 104, FinancialAccountID: ptr.Int64(11), CurrencyID: 1, Name: "Principal", Kind: model.LedgerAccountLiability}
	expenses       = &model.LedgerAccount{ID: 200, CurrencyID: 1, Name: "Expenses", Kind: model.LedgerAccountExpense}
	expensesUSD    = &model.LedgerAccount{ID: 201, CurrencyID: 2, Name: "Expenses", Kind: model.LedgerAccountExpense}
	income         = &model.LedgerAccount{ID: 210, CurrencyID: 1, Name: "Income", Kind: model.LedgerAccountIncome}
	incomeUSD      = &model.LedgerAccount{ID: 211, CurrencyID: 2, Name: "Income", Kind: model.LedgerAccountIncome}
	openingBalance = &model.LedgerAccount{ID: 300, CurrencyID: 1, Name: "Opening balances", Kind: model.LedgerAccountEquity}
	fxClearing     = &model.LedgerAccount{ID: 301, CurrencyID: 1, Name: "FX clearing", Kind: model.LedgerAccountAsset}
	fxClearingUSD  = &model.LedgerAccount{ID: 302, CurrencyID: 2, Name: "FX clearing", Kind: model.LedgerAccountAsset}
)

// The categories an expense or an income names, as RetrieveCategory reports
// them. A closed one is still a category of an entry.
var (
	groceries = &model.Category{ID: 20, Name: "Groceries", Kind: model.LedgerAccountExpense}
	oldHobby  = &model.Category{ID: 21, Name: "Old hobby", Kind: model.LedgerAccountExpense, ClosedAt: ptr.NewPtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))}
	interest  = &model.Category{ID: 22, Name: "Mortgage interest", Kind: model.LedgerAccountExpense}
	salary    = &model.Category{ID: 30, Name: "Salary", Kind: model.LedgerAccountIncome}
)

// mockLedgerAccounts lets RetrieveLedgerAccount find every fixture account,
// and CreateLedgerAccount find the "Expenses" and "Income" account of each
// fixture currency. The mocks are optional, as each case uses only some.
func mockLedgerAccounts(env *testsuite.TestWorkflowEnvironment) {
	shared := []*model.LedgerAccount{expenses, expensesUSD, income, incomeUSD}
	for _, account := range append([]*model.LedgerAccount{cash, cardBalance, savingsVault, cashUSD, principal, openingBalance, fxClearing, fxClearingUSD}, shared...) {
		env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, act.RetrieveLedgerAccountParams{ID: account.ID}).
			Return(&act.RetrieveLedgerAccountResult{LedgerAccount: account}, nil).Maybe()
	}
	for _, account := range shared {
		env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, act.CreateLedgerAccountInput{
			CurrencyID: account.CurrencyID, Name: account.Name, Kind: account.Kind,
		}).Return(&act.CreateLedgerAccountResult{LedgerAccountID: account.ID}, nil).Maybe()
	}
}

// mockCategories lets RetrieveCategory find every fixture category.
func mockCategories(env *testsuite.TestWorkflowEnvironment) {
	for _, category := range []*model.Category{groceries, oldHobby, interest, salary} {
		env.OnWorkflow(categorieswf.RetrieveCategory, mock.Anything, ct.RetrieveCategoryParams{ID: category.ID}).
			Return(&ct.RetrieveCategoryResult{Category: category}, nil).Maybe()
	}
}

// samePostings reports whether got books what want does, on the same accounts
// and with the same categories.
func samePostings(got, want []jt.PostingInput) bool {
	if len(got) != len(want) {
		return false
	}
	for i, posting := range want {
		sameCategory := (got[i].CategoryID == nil) == (posting.CategoryID == nil) &&
			(posting.CategoryID == nil || *got[i].CategoryID == *posting.CategoryID)
		if got[i].LedgerAccountID != posting.LedgerAccountID || got[i].Side != posting.Side ||
			!got[i].Amount.Equal(posting.Amount) || !sameCategory {
			return false
		}
	}
	return true
}

// entry matches the journal entry want by what it books rather than by its
// representation: a decimal such as 12.50 arrives as 12.5, equal in value but
// not in its fields.
func entry(want jt.PostJournalEntryInput) any {
	return mock.MatchedBy(func(got jt.PostJournalEntryInput) bool {
		sameDescription := (got.Description == nil) == (want.Description == nil) &&
			(got.Description == nil || *got.Description == *want.Description)
		if got.IdempotencyKey != want.IdempotencyKey || !got.OccurredAt.Equal(want.OccurredAt) || !sameDescription ||
			got.CounterpartyID != nil || len(got.Metadata) != 0 || !slices.Equal(got.TagIDs, want.TagIDs) {
			return false
		}
		return samePostings(got.Postings, want.Postings)
	})
}

// replacement matches the replace want by what it books, as entry does, and
// checks that it carries no position movement.
func replacement(want jt.ReplaceJournalEntryInput) any {
	return mock.MatchedBy(func(got jt.ReplaceJournalEntryInput) bool {
		sameDescription := (got.Description == nil) == (want.Description == nil) &&
			(got.Description == nil || *got.Description == *want.Description)
		if got.JournalEntryID != want.JournalEntryID || !got.OccurredAt.Equal(want.OccurredAt) || !sameDescription ||
			!maps.Equal(got.Metadata, want.Metadata) || len(got.Movements) != 0 || !slices.Equal(got.TagIDs, want.TagIDs) {
			return false
		}
		return samePostings(got.Postings, want.Postings)
	})
}

// soldSince is a holding an update left negative, as ReplaceJournalEntry
// reports it.
var soldSince = []lt.NegativeHolding{{FinancialAccountID: 10, FinancialAccountName: "Revolut Personal", AssetID: 7, AssetName: "ACME", Units: decimal.NewFromInt(-3)}}
