package accountsetup

import (
	"errors"
	"strings"
	"time"

	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

var (
	euro   = &model.Currency{ID: 1, Code: "EUR"}
	dollar = &model.Currency{ID: 2, Code: "USD"}
)

// openAccountCase describes what the child workflows find and create for one
// OpenAccount request, and what the request returns.
type openAccountCase struct {
	input ast.OpenAccountInput
	// institution is the institution the request belongs to; existing says
	// whether it is found by name or inserted.
	institution *model.Institution
	existing    bool
	account     *model.FinancialAccount
	// ledgerAccounts are the balances and shared accounts, in the order the
	// workflow creates them, each with the CreateLedgerAccount input it
	// expects.
	ledgerAccounts []expectedLedgerAccount
	// opening is the opening entry the workflow posts, if any.
	opening *lt.PostJournalEntryInput
	want    *ast.OpenAccountResult
}

type expectedLedgerAccount struct {
	input   lt.CreateLedgerAccountInput
	account *model.LedgerAccount
}

func (tc openAccountCase) mock(env *testsuite.TestWorkflowEnvironment) {
	known := map[string]*model.Currency{"EUR": euro, "USD": dollar}
	for _, currency := range tc.input.Currencies {
		code := strings.ToUpper(currency.Code)
		env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, &ct.RetrieveCurrencyParams{Code: code}).Return(known[code], nil)
	}
	if tc.existing {
		env.OnWorkflow(institutionwf.RetrieveInstitution, mock.Anything, it.RetrieveInstitutionParams{Name: tc.institution.Name}).
			Return(&it.RetrieveInstitutionResult{Institution: tc.institution}, nil)
	} else {
		env.OnWorkflow(institutionwf.RetrieveInstitution, mock.Anything, it.RetrieveInstitutionParams{Name: tc.institution.Name}).
			Return(nil, internalerrors.ErrNotFound{ResourceType: "Institution"})
		env.OnWorkflow(institutionwf.InsertInstitution, mock.Anything, it.InsertInstitutionParams{Name: tc.institution.Name, Kind: tc.institution.Kind}).
			Return(&it.InsertInstitutionResult{ID: tc.institution.ID}, nil).Once()
	}
	env.OnWorkflow(institutionwf.RetrieveInstitution, mock.Anything, it.RetrieveInstitutionParams{ID: tc.institution.ID}).
		Return(&it.RetrieveInstitutionResult{Institution: tc.institution}, nil)

	env.OnWorkflow(ledgeraccountswf.CreateFinancialAccount, mock.Anything, lt.CreateFinancialAccountInput{
		InstitutionID: tc.institution.ID, Name: tc.account.Name,
	}).Return(&lt.CreateFinancialAccountResult{FinancialAccountID: tc.account.ID}, nil)
	env.OnWorkflow(ledgeraccountswf.RetrieveFinancialAccount, mock.Anything, lt.RetrieveFinancialAccountParams{ID: tc.account.ID}).
		Return(&lt.RetrieveFinancialAccountResult{FinancialAccount: tc.account}, nil)

	for _, expected := range tc.ledgerAccounts {
		env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, expected.input).
			Return(&lt.CreateLedgerAccountResult{LedgerAccountID: expected.account.ID}, nil)
		env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: expected.account.ID}).
			Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: expected.account}, nil)
	}

	if tc.opening != nil {
		env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, openingEntry(*tc.opening)).
			Return(&lt.PostJournalEntryResult{JournalEntryID: 500}, nil).Once()
	}
}

// balance is a balance under financialAccount, as both the child workflows
// and the result describe it.
func balance(id int64, financialAccount *model.FinancialAccount, currency *model.Currency, name string, kind model.LedgerAccountKind) expectedLedgerAccount {
	return expectedLedgerAccount{
		input:   lt.CreateLedgerAccountInput{FinancialAccountID: ptr.Int64(financialAccount.ID), CurrencyID: currency.ID, Name: name, Kind: kind},
		account: &model.LedgerAccount{ID: id, FinancialAccountID: ptr.Int64(financialAccount.ID), CurrencyID: currency.ID, Currency: currency, Name: name, Kind: kind},
	}
}

// shared are the shared accounts of currency, "Opening balances" with the id
// given and "FX clearing" with the next.
func shared(id int64, currency *model.Currency) []expectedLedgerAccount {
	return []expectedLedgerAccount{
		{
			input:   lt.CreateLedgerAccountInput{CurrencyID: currency.ID, Name: "Opening balances", Kind: model.LedgerAccountEquity},
			account: &model.LedgerAccount{ID: id, CurrencyID: currency.ID, Currency: currency, Name: "Opening balances", Kind: model.LedgerAccountEquity},
		},
		{
			input:   lt.CreateLedgerAccountInput{CurrencyID: currency.ID, Name: "FX clearing", Kind: model.LedgerAccountAsset},
			account: &model.LedgerAccount{ID: id + 1, CurrencyID: currency.ID, Currency: currency, Name: "FX clearing", Kind: model.LedgerAccountAsset},
		},
	}
}

// rows are the result rows of the expected ledger accounts.
func rows(institution *model.Institution, financialAccount *model.FinancialAccount, expected []expectedLedgerAccount) []lt.AccountSummary {
	summaries := make([]lt.AccountSummary, 0, len(expected))
	for _, e := range expected {
		summary := lt.AccountSummary{
			LedgerAccountID: e.account.ID, LedgerAccountName: e.account.Name, Kind: e.account.Kind,
			CurrencyID: e.account.CurrencyID, CurrencyCode: e.account.Currency.Code,
		}
		if e.account.FinancialAccountID != nil {
			summary.FinancialAccountID = ptr.Int64(financialAccount.ID)
			summary.FinancialAccountName = financialAccount.Name
			summary.InstitutionName = institution.Name
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func concat(groups ...[]expectedLedgerAccount) []expectedLedgerAccount {
	var all []expectedLedgerAccount
	for _, group := range groups {
		all = append(all, group...)
	}
	return all
}

// TestOpenAccount checks, for each account type, that the institution is
// found or created, the financial account is created by its name or the
// default one, every currency gets the type's balances and the shared
// accounts, and the opening balance is booked on the type's main balance.
func (s *WorkflowUnitTestSuite) TestOpenAccount() {
	start := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	revolut := &model.Institution{ID: 2, Name: "Revolut", Kind: "bank"}
	personal := &model.FinancialAccount{ID: 10, InstitutionID: 2, Name: "Revolut Personal"}
	ibkr := &model.Institution{ID: 3, Name: "IBKR", Kind: "broker"}
	ibkrAccount := &model.FinancialAccount{ID: 11, InstitutionID: 3, Name: "IBKR"}
	amex := &model.Institution{ID: 4, Name: "Amex", Kind: "card issuer"}
	amexGold := &model.FinancialAccount{ID: 12, InstitutionID: 4, Name: "Amex Gold"}
	physicalCash := &model.Institution{ID: 5, Name: "Physical cash", Kind: "cash"}
	wallet := &model.FinancialAccount{ID: 13, InstitutionID: 5, Name: "Wallet"}

	revolutLedger := concat([]expectedLedgerAccount{balance(100, personal, euro, "Cash", model.LedgerAccountAsset)}, shared(200, euro))
	ibkrLedger := concat(
		[]expectedLedgerAccount{balance(110, ibkrAccount, euro, "Cash", model.LedgerAccountAsset), balance(111, ibkrAccount, euro, "Investments", model.LedgerAccountAsset)},
		shared(200, euro),
		[]expectedLedgerAccount{balance(112, ibkrAccount, dollar, "Cash", model.LedgerAccountAsset), balance(113, ibkrAccount, dollar, "Investments", model.LedgerAccountAsset)},
		shared(210, dollar),
	)
	amexLedger := concat([]expectedLedgerAccount{balance(120, amexGold, euro, "Card balance", model.LedgerAccountLiability)}, shared(200, euro))
	walletLedger := concat([]expectedLedgerAccount{balance(130, wallet, euro, "Cash", model.LedgerAccountAsset)}, shared(200, euro))

	tests := map[string]openAccountCase{
		"a bank account at an existing institution, with an opening balance at the time of the request": {
			input: ast.OpenAccountInput{
				Type: ast.AccountTypeBank, InstitutionName: "Revolut", AccountName: "Revolut Personal",
				Currencies: []ast.CurrencyBalance{{Code: "EUR", OpeningBalance: decimal.NewFromInt(1200)}},
			},
			institution: revolut, existing: true, account: personal, ledgerAccounts: revolutLedger,
			opening: &lt.PostJournalEntryInput{
				IdempotencyKey: "opening-balance:100", OccurredAt: start, Description: ptr.NewPtr("Opening balance"),
				Postings: []lt.PostingInput{
					{LedgerAccountID: 100, Amount: decimal.NewFromInt(1200), Side: model.PostingDebit},
					{LedgerAccountID: 200, Amount: decimal.NewFromInt(1200), Side: model.PostingCredit},
				},
			},
			want: &ast.OpenAccountResult{
				Outcome: ast.OutcomeOpened, InstitutionID: 2, InstitutionName: "Revolut", FinancialAccountID: 10, FinancialAccountName: "Revolut Personal",
				Balances: ast.Balances{Balances: rows(revolut, personal, revolutLedger), OpeningEntries: []ast.OpeningEntry{{LedgerAccountID: 100, JournalEntryID: 500}}},
			},
		},
		"a bank account at a new institution, once confirmed": {
			input: ast.OpenAccountInput{
				Type: ast.AccountTypeBank, InstitutionName: "Revolut", AccountName: "Revolut Personal", CreateInstitution: true,
				Currencies: []ast.CurrencyBalance{{Code: "EUR"}},
			},
			institution: revolut, account: personal, ledgerAccounts: revolutLedger,
			want: &ast.OpenAccountResult{
				Outcome: ast.OutcomeOpened, InstitutionID: 2, InstitutionName: "Revolut", FinancialAccountID: 10, FinancialAccountName: "Revolut Personal",
				Balances: ast.Balances{Balances: rows(revolut, personal, revolutLedger)},
			},
		},
		"a brokerage in two currencies, named after its institution": {
			input: ast.OpenAccountInput{
				Type: ast.AccountTypeBrokerage, InstitutionName: "IBKR",
				Currencies: []ast.CurrencyBalance{{Code: "EUR"}, {Code: "usd"}},
			},
			institution: ibkr, existing: true, account: ibkrAccount, ledgerAccounts: ibkrLedger,
			want: &ast.OpenAccountResult{
				Outcome: ast.OutcomeOpened, InstitutionID: 3, InstitutionName: "IBKR", FinancialAccountID: 11, FinancialAccountName: "IBKR",
				Balances: ast.Balances{Balances: rows(ibkr, ibkrAccount, ibkrLedger)},
			},
		},
		"a card with the amount owed, at a given instant": {
			input: ast.OpenAccountInput{
				Type: ast.AccountTypeCard, InstitutionName: "Amex", AccountName: "Amex Gold", CreateInstitution: true,
				Currencies:        []ast.CurrencyBalance{{Code: "EUR", OpeningBalance: decimal.NewFromInt(300)}},
				OpeningBalancesAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			institution: amex, account: amexGold, ledgerAccounts: amexLedger,
			opening: &lt.PostJournalEntryInput{
				IdempotencyKey: "opening-balance:120", OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Description: ptr.NewPtr("Opening balance"),
				Postings: []lt.PostingInput{
					{LedgerAccountID: 120, Amount: decimal.NewFromInt(300), Side: model.PostingCredit},
					{LedgerAccountID: 200, Amount: decimal.NewFromInt(300), Side: model.PostingDebit},
				},
			},
			want: &ast.OpenAccountResult{
				Outcome: ast.OutcomeOpened, InstitutionID: 4, InstitutionName: "Amex", FinancialAccountID: 12, FinancialAccountName: "Amex Gold",
				Balances: ast.Balances{Balances: rows(amex, amexGold, amexLedger), OpeningEntries: []ast.OpeningEntry{{LedgerAccountID: 120, JournalEntryID: 500}}},
			},
		},
		"a wallet, whose institution is created without asking": {
			input:       ast.OpenAccountInput{Type: ast.AccountTypeCash, Currencies: []ast.CurrencyBalance{{Code: "EUR"}}},
			institution: physicalCash, account: wallet, ledgerAccounts: walletLedger,
			want: &ast.OpenAccountResult{
				Outcome: ast.OutcomeOpened, InstitutionID: 5, InstitutionName: "Physical cash", FinancialAccountID: 13, FinancialAccountName: "Wallet",
				Balances: ast.Balances{Balances: rows(physicalCash, wallet, walletLedger)},
			},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			env.SetStartTime(start)
			tc.mock(env)
			env.ExecuteWorkflow(OpenAccount, tc.input)

			require.NoError(s.T(), env.GetWorkflowError())
			var got *ast.OpenAccountResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), tc.want, got)
			env.AssertExpectations(s.T())
		})
	}
}

// TestOpenAccount_InstitutionNotFound checks that a missing institution is not
// created without confirmation: the request returns institution_not_found and
// writes nothing, since no other child workflow is mocked.
func (s *WorkflowUnitTestSuite) TestOpenAccount_InstitutionNotFound() {
	env := s.NewTestWorkflowEnvironment()
	env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, mock.Anything).Return(euro, nil)
	env.OnWorkflow(institutionwf.RetrieveInstitution, mock.Anything, it.RetrieveInstitutionParams{Name: "Revolt"}).
		Return(nil, internalerrors.ErrNotFound{ResourceType: "Institution"})

	env.ExecuteWorkflow(OpenAccount, ast.OpenAccountInput{
		Type: ast.AccountTypeBank, InstitutionName: "Revolt", Currencies: []ast.CurrencyBalance{{Code: "EUR"}},
	})

	require.NoError(s.T(), env.GetWorkflowError())
	var got *ast.OpenAccountResult
	require.NoError(s.T(), env.GetWorkflowResult(&got))
	require.Equal(s.T(), &ast.OpenAccountResult{Outcome: ast.OutcomeInstitutionNotFound, InstitutionName: "Revolt"}, got)
}

// TestOpenAccount_Errors checks that an invalid request, an unknown currency,
// or a closed institution or account fails before any ledger account is
// created, since none of those child workflows is mocked, and that a lookup
// that fails for another reason than a missing institution is an error.
func (s *WorkflowUnitTestSuite) TestOpenAccount_Errors() {
	closedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	revolut := &model.Institution{ID: 2, Name: "Revolut", Kind: "bank"}
	input := ast.OpenAccountInput{Type: ast.AccountTypeBank, InstitutionName: "Revolut", Currencies: []ast.CurrencyBalance{{Code: "EUR"}, {Code: "XYZ"}}}

	tests := map[string]struct {
		input     ast.OpenAccountInput
		mock      func(env *testsuite.TestWorkflowEnvironment)
		wantError string
	}{
		"an invalid request": {
			input:     ast.OpenAccountInput{Type: ast.AccountTypeBank, Currencies: []ast.CurrencyBalance{{Code: "EUR"}}},
			mock:      func(*testsuite.TestWorkflowEnvironment) {},
			wantError: "the institution is required",
		},
		"an unknown currency, after a known one": {
			input: input,
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, &ct.RetrieveCurrencyParams{Code: "EUR"}).Return(euro, nil)
				env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, &ct.RetrieveCurrencyParams{Code: "XYZ"}).
					Return(nil, internalerrors.ErrNotFound{ResourceType: "Currency", ID: "XYZ"})
			},
			wantError: "retrieve currency XYZ",
		},
		"a closed institution": {
			input: ast.OpenAccountInput{Type: ast.AccountTypeBank, InstitutionName: "Revolut", Currencies: []ast.CurrencyBalance{{Code: "EUR"}}},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, mock.Anything).Return(euro, nil)
				closed := *revolut
				closed.ClosedAt = &closedAt
				env.OnWorkflow(institutionwf.RetrieveInstitution, mock.Anything, mock.Anything).
					Return(&it.RetrieveInstitutionResult{Institution: &closed}, nil)
			},
			wantError: "resource Institution with ID: 2 is closed",
		},
		"looking the institution up fails": {
			input: ast.OpenAccountInput{Type: ast.AccountTypeBank, InstitutionName: "Revolut", Currencies: []ast.CurrencyBalance{{Code: "EUR"}}},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, mock.Anything).Return(euro, nil)
				env.OnWorkflow(institutionwf.RetrieveInstitution, mock.Anything, mock.Anything).Return(nil, errors.New("database down"))
			},
			wantError: "retrieve institution Revolut",
		},
		"a closed financial account": {
			input: ast.OpenAccountInput{Type: ast.AccountTypeBank, InstitutionName: "Revolut", AccountName: "Revolut Old", Currencies: []ast.CurrencyBalance{{Code: "EUR"}}},
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, mock.Anything).Return(euro, nil)
				env.OnWorkflow(institutionwf.RetrieveInstitution, mock.Anything, mock.Anything).
					Return(&it.RetrieveInstitutionResult{Institution: revolut}, nil)
				env.OnWorkflow(ledgeraccountswf.CreateFinancialAccount, mock.Anything, mock.Anything).
					Return(&lt.CreateFinancialAccountResult{FinancialAccountID: 14}, nil)
				env.OnWorkflow(ledgeraccountswf.RetrieveFinancialAccount, mock.Anything, mock.Anything).
					Return(&lt.RetrieveFinancialAccountResult{FinancialAccount: &model.FinancialAccount{ID: 14, InstitutionID: 2, Name: "Revolut Old", ClosedAt: &closedAt}}, nil)
			},
			wantError: "resource FinancialAccount with ID: 14 is closed",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			tc.mock(env)
			env.ExecuteWorkflow(OpenAccount, tc.input)

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
