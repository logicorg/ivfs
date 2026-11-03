package setup

import (
	"fmt"
	"time"

	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows/facade/accountsetup"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Test_OpenAccount opens accounts the way a client does, through the account
// setup facade (ADR 0006), against the real workflows, activities, and
// database. It checks what the mocked tests cannot: that a missing institution
// is recognised through the real chain of child workflow errors and nothing is
// written, that repeating a request changes nothing and posts no second
// opening entry, that adding a currency leaves the others alone, and that
// ListAccounts and the balances see what was opened.
func (s *UnitTestSuite) Test_OpenAccount() {
	temporaltesting.Insert(s.T(), s.DB, testhelpers.USDollar())
	openedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	revolut := ast.OpenAccountInput{
		Type: ast.AccountTypeBank, InstitutionName: "Revolut", AccountName: "Revolut Personal",
		Currencies:        []ast.CurrencyBalance{{Code: "EUR", OpeningBalance: decimal.NewFromInt(1200)}},
		OpeningBalancesAt: openedAt,
	}

	// A new institution is not created until the request confirms it.

	var notFound ast.OpenAccountResult
	s.runWorkflow(accountsetup.OpenAccount, revolut, &notFound)
	require.Equal(s.T(), ast.OpenAccountResult{Outcome: ast.OutcomeInstitutionNotFound, InstitutionName: "Revolut"}, notFound)
	s.requireCount((*model.Institution)(nil), 0)
	s.requireCount((*model.FinancialAccount)(nil), 0)
	s.requireCount((*model.LedgerAccount)(nil), 0)

	revolut.CreateInstitution = true
	var opened ast.OpenAccountResult
	s.runWorkflow(accountsetup.OpenAccount, revolut, &opened)
	require.Equal(s.T(), ast.OutcomeOpened, opened.Outcome)
	require.Equal(s.T(), []string{
		"Revolut / Revolut Personal / Cash EUR",
		"Opening balances EUR",
		"FX clearing EUR",
	}, describe(opened.Balances.Balances))
	require.Len(s.T(), opened.OpeningEntries, 1)

	// Opening the same account again changes nothing.

	var reopened ast.OpenAccountResult
	s.runWorkflow(accountsetup.OpenAccount, revolut, &reopened)
	require.Equal(s.T(), opened, reopened)
	s.requireCount((*model.JournalEntry)(nil), 1)

	// Opening it with another currency adds that currency's balances only.

	var withDollars ast.OpenAccountResult
	s.runWorkflow(accountsetup.OpenAccount, ast.OpenAccountInput{
		Type: ast.AccountTypeBank, InstitutionName: "Revolut", AccountName: "Revolut Personal",
		Currencies: []ast.CurrencyBalance{{Code: "usd"}},
	}, &withDollars)
	require.Equal(s.T(), opened.FinancialAccountID, withDollars.FinancialAccountID)
	require.Equal(s.T(), []string{
		"Revolut / Revolut Personal / Cash USD",
		"Opening balances USD",
		"FX clearing USD",
	}, describe(withDollars.Balances.Balances))
	require.Empty(s.T(), withDollars.OpeningEntries)

	// A savings vault holds its opening balance once, however often it is
	// added.

	vault := ast.AddBalanceInput{
		FinancialAccountID: opened.FinancialAccountID, Name: "Savings vault", CurrencyCode: "EUR",
		OpeningBalance: decimal.NewFromInt(500), OpeningBalanceAt: openedAt,
	}
	var added, addedAgain ast.AddBalanceResult
	s.runWorkflow(accountsetup.AddBalance, vault, &added)
	s.runWorkflow(accountsetup.AddBalance, vault, &addedAgain)
	require.Equal(s.T(), added, addedAgain)
	require.Len(s.T(), added.OpeningEntries, 1)
	s.requireCount((*model.JournalEntry)(nil), 2)

	// A wallet needs no institution: "Physical cash" is created on first use.

	var wallet ast.OpenAccountResult
	s.runWorkflow(accountsetup.OpenAccount, ast.OpenAccountInput{
		Type:              ast.AccountTypeCash,
		Currencies:        []ast.CurrencyBalance{{Code: "EUR", OpeningBalance: decimal.NewFromInt(80)}},
		OpeningBalancesAt: openedAt,
	}, &wallet)
	require.Equal(s.T(), []string{
		"Physical cash / Wallet / Cash EUR",
		"Opening balances EUR",
		"FX clearing EUR",
	}, describe(wallet.Balances.Balances))

	// Everything opened is listed once, and the opening balances add up.

	var listed lt.ListAccountsResult
	s.runWorkflow(ledgeraccountswf.ListAccounts, lt.ListAccountsParams{}, &listed)
	require.ElementsMatch(s.T(), []string{
		"Revolut / Revolut Personal / Cash EUR",
		"Revolut / Revolut Personal / Cash USD",
		"Revolut / Revolut Personal / Savings vault EUR",
		"Physical cash / Wallet / Cash EUR",
		"Opening balances EUR",
		"FX clearing EUR",
		"Opening balances USD",
		"FX clearing USD",
	}, describe(listed.Accounts))

	s.requireBalance(opened.Balances.Balances[0].LedgerAccountID, 1200)
	s.requireBalance(added.LedgerAccountID, 500)
	s.requireBalance(wallet.Balances.Balances[0].LedgerAccountID, 80)
	s.requireBalance(opened.Balances.Balances[1].LedgerAccountID, 1200+500+80)
}

// describe names each row the way a person would read it: the institution,
// the financial account, and the balance in its currency, or only the
// balance for a shared account.
func describe(rows []lt.AccountSummary) []string {
	described := make([]string, 0, len(rows))
	for _, row := range rows {
		name := fmt.Sprintf("%s %s", row.LedgerAccountName, row.CurrencyCode)
		if row.FinancialAccountID != nil {
			name = fmt.Sprintf("%s / %s / %s", row.InstitutionName, row.FinancialAccountName, name)
		}
		described = append(described, name)
	}
	return described
}

func (s *UnitTestSuite) requireCount(m any, want int) {
	s.T().Helper()

	count, err := s.DB.NewSelect().Model(m).Count(s.T().Context())
	require.NoError(s.T(), err)
	require.Equalf(s.T(), want, count, "%T rows", m)
}

// requireBalance checks a ledger account's balance, in its natural direction.
func (s *UnitTestSuite) requireBalance(ledgerAccountID int64, want int64) {
	s.T().Helper()

	var balance lt.LedgerAccountBalanceResult
	s.runWorkflow(reportingwf.LedgerAccountBalance, lt.LedgerAccountBalanceParams{LedgerAccountID: ledgerAccountID}, &balance)
	require.Truef(s.T(), balance.Balance.Equal(decimal.NewFromInt(want)),
		"%s balance %s, want %d", balance.LedgerAccountName, balance.Balance, want)
}
