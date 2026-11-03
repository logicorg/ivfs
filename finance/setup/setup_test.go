// Package setup shows how the account-setup workflows compose into a ledger
// a client can actually use.
package setup

import (
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/registration"
	"github.com/logicorg/ivfs/finance/workflows/facade/categories"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite

	eur *model.Currency
}

func (s *UnitTestSuite) BeforeTest(_, _ string) {
	sqllib.InitDatabaseT(s.T(), s.DB)

	s.eur = testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, s.eur)
}

func (s *UnitTestSuite) AfterTest(_, _ string) { s.DropAllTables() }

func Test_UnitTestSuite(t *testing.T) { suite.Run(t, new(UnitTestSuite)) }

// Test_PersonalSetup builds the account structure a new user needs before
// recording anything: the institution they hold an account with, a financial
// account carrying a cash ledger account, an equity account for opening
// balances, and one income and one expense category. It drives the client
// facing workflows against the real activities and database, then re-derives
// every identifier through the retrieve and list workflows, which is what a
// client has to be able to do to use the backend without reading the
// database directly.
func (s *UnitTestSuite) Test_PersonalSetup() {
	var params any

	// Create the Revolut institution and FinancialAccount

	params = it.InsertInstitutionParams{Name: "Revolut", Kind: "bank"}
	var institution it.InsertInstitutionResult
	s.runWorkflow(institutionwf.InsertInstitution, params, &institution)

	// Passing a cash currency also creates the account's first cash ledger
	// account, in the same transaction.
	params = lt.CreateFinancialAccountInput{
		InstitutionID:  institution.ID,
		Name:           "Revolut EUR",
		CashCurrencyID: ptr.Int64(s.eur.ID),
	}
	var account lt.CreateFinancialAccountResult
	s.runWorkflow(ledgeraccountswf.CreateFinancialAccount, params, &account)
	require.NotNil(s.T(), account.CashLedgerAccountID)

	// Create Ledger accounts and categories

	// The category facade covers income and expense only, so equity goes
	// through the generic ledger account workflow.

	params = lt.CreateLedgerAccountInput{
		CurrencyID: s.eur.ID,
		Name:       "Opening Balances",
		Kind:       model.LedgerAccountEquity,
	}
	var openingBalances lt.CreateLedgerAccountResult
	s.runWorkflow(ledgeraccountswf.CreateLedgerAccount, params, &openingBalances)

	params = ct.CreateCategoryInput{
		CurrencyID: s.eur.ID,
		Name:       "Salary",
		Kind:       model.LedgerAccountIncome,
	}
	var salary ct.CategoryResult
	s.runWorkflow(categories.CreateCategory, params, &salary)

	params = ct.CreateCategoryInput{
		CurrencyID: s.eur.ID,
		Name:       "Groceries",
		Kind:       model.LedgerAccountExpense,
	}
	var groceries ct.CategoryResult
	s.runWorkflow(categories.CreateCategory, params, &groceries)

	// Perform some gets to confirm workflows can retrieve information correctly

	params = it.RetrieveInstitutionParams{ID: institution.ID}
	var retrieved it.RetrieveInstitutionResult
	s.runWorkflow(institutionwf.RetrieveInstitution, params, &retrieved)
	require.Equal(s.T(), "Revolut", retrieved.Institution.Name)

	params = lt.ListFinancialAccountsParams{InstitutionID: ptr.Int64(institution.ID)}
	var financialAccounts lt.ListFinancialAccountsResult
	s.runWorkflow(ledgeraccountswf.ListFinancialAccounts, params, &financialAccounts)
	require.Len(s.T(), financialAccounts.FinancialAccounts, 1)
	require.Equal(s.T(), account.FinancialAccountID, financialAccounts.FinancialAccounts[0].ID)

	params = lt.ListLedgerAccountsParams{FinancialAccountID: ptr.Int64(account.FinancialAccountID)}
	var cashAccounts lt.ListLedgerAccountsResult
	s.runWorkflow(ledgeraccountswf.ListLedgerAccounts, params, &cashAccounts)
	require.Len(s.T(), cashAccounts.LedgerAccounts, 1)
	require.Equal(s.T(), *account.CashLedgerAccountID, cashAccounts.LedgerAccounts[0].ID)

	// Equity and category accounts are not linked to a financial account, so
	// they only show up in the unfiltered chart of accounts.
	var chartOfAccounts lt.ListLedgerAccountsResult
	s.runWorkflow(ledgeraccountswf.ListLedgerAccounts, lt.ListLedgerAccountsParams{}, &chartOfAccounts)

	chartIDs := make([]int64, 0, len(chartOfAccounts.LedgerAccounts))
	for _, ledgerAccount := range chartOfAccounts.LedgerAccounts {
		chartIDs = append(chartIDs, ledgerAccount.ID)
	}
	require.ElementsMatch(s.T(), []int64{
		*account.CashLedgerAccountID,
		openingBalances.LedgerAccountID,
		salary.LedgerAccountID,
		groceries.LedgerAccountID,
	}, chartIDs)
}

// runWorkflow executes wf on a fresh environment wired to the real activities
// rather than mocks, so each call exercises the whole path down to the
// database. Everything the worker registers is registered, so a facade finds
// the workflows it runs as children. result may be nil for workflows that only
// return an error.
func (s *UnitTestSuite) runWorkflow(wf, input, result any) {
	s.T().Helper()

	env := s.NewTestWorkflowEnvironment()
	registration.Register(env, tenancy.Static(s.DB))

	env.ExecuteWorkflow(wf, input)
	require.NoError(s.T(), env.GetWorkflowError())

	if result != nil {
		require.NoError(s.T(), env.GetWorkflowResult(result))
	}
}
