package transactions

import (
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
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
// RetrieveLedgerAccount reports them: the balances of Revolut Personal (10),
// its categories, and the shared accounts, which belong to no financial
// account.
var (
	cash           = &model.LedgerAccount{ID: 100, FinancialAccountID: ptr.Int64(10), CurrencyID: 1, Name: "Cash", Kind: model.LedgerAccountAsset}
	cardBalance    = &model.LedgerAccount{ID: 101, FinancialAccountID: ptr.Int64(10), CurrencyID: 1, Name: "Card balance", Kind: model.LedgerAccountLiability}
	savingsVault   = &model.LedgerAccount{ID: 102, FinancialAccountID: ptr.Int64(10), CurrencyID: 1, Name: "Savings vault", Kind: model.LedgerAccountAsset}
	cashUSD        = &model.LedgerAccount{ID: 103, FinancialAccountID: ptr.Int64(10), CurrencyID: 2, Name: "Cash (USD)", Kind: model.LedgerAccountAsset}
	groceries      = &model.LedgerAccount{ID: 200, CurrencyID: 1, Name: "Groceries", Kind: model.LedgerAccountExpense}
	groceriesUSD   = &model.LedgerAccount{ID: 201, CurrencyID: 2, Name: "Groceries (USD)", Kind: model.LedgerAccountExpense}
	salary         = &model.LedgerAccount{ID: 210, CurrencyID: 1, Name: "Salary", Kind: model.LedgerAccountIncome}
	salaryUSD      = &model.LedgerAccount{ID: 211, CurrencyID: 2, Name: "Salary (USD)", Kind: model.LedgerAccountIncome}
	openingBalance = &model.LedgerAccount{ID: 300, CurrencyID: 1, Name: "Opening balances", Kind: model.LedgerAccountEquity}
	fxClearing     = &model.LedgerAccount{ID: 301, CurrencyID: 1, Name: "FX clearing", Kind: model.LedgerAccountAsset}
	fxClearingUSD  = &model.LedgerAccount{ID: 302, CurrencyID: 2, Name: "FX clearing", Kind: model.LedgerAccountAsset}
)

// mockLedgerAccounts lets RetrieveLedgerAccount find every fixture account.
// The mocks are optional, as each case retrieves only two.
func mockLedgerAccounts(env *testsuite.TestWorkflowEnvironment) {
	for _, account := range []*model.LedgerAccount{
		cash, cardBalance, savingsVault, cashUSD, groceries, groceriesUSD, salary, salaryUSD, openingBalance, fxClearing, fxClearingUSD,
	} {
		env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: account.ID}).
			Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: account}, nil).Maybe()
	}
}

// entry matches the journal entry want by what it books rather than by its
// representation: a decimal such as 12.50 arrives as 12.5, equal in value but
// not in its fields.
func entry(want lt.PostJournalEntryInput) any {
	return mock.MatchedBy(func(got lt.PostJournalEntryInput) bool {
		sameDescription := (got.Description == nil) == (want.Description == nil) &&
			(got.Description == nil || *got.Description == *want.Description)
		if got.IdempotencyKey != want.IdempotencyKey || !got.OccurredAt.Equal(want.OccurredAt) || !sameDescription ||
			got.ReversalOfEntryID != nil || got.CounterpartyID != nil || len(got.Metadata) != 0 || len(got.Postings) != len(want.Postings) {
			return false
		}
		for i, posting := range want.Postings {
			if got.Postings[i].LedgerAccountID != posting.LedgerAccountID || got.Postings[i].Side != posting.Side ||
				!got.Postings[i].Amount.Equal(posting.Amount) {
				return false
			}
		}
		return true
	})
}
