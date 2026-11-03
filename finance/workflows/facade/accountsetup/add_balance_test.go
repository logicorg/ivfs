package accountsetup

import (
	"errors"
	"time"

	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// addBalanceFixtures is a savings vault added to Revolut Personal, in EUR, as
// the child workflows report it.
type addBalanceFixtures struct {
	euro            *model.Currency
	institution     *model.Institution
	account         *model.FinancialAccount
	vault           *model.LedgerAccount
	openingBalances *model.LedgerAccount
	fxClearing      *model.LedgerAccount
}

func newAddBalanceFixtures() addBalanceFixtures {
	euro := &model.Currency{ID: 1, Code: "EUR"}
	return addBalanceFixtures{
		euro:            euro,
		institution:     &model.Institution{ID: 2, Name: "Revolut"},
		account:         &model.FinancialAccount{ID: 10, InstitutionID: 2, Name: "Revolut Personal"},
		vault:           &model.LedgerAccount{ID: 100, FinancialAccountID: ptr.Int64(10), CurrencyID: 1, Currency: euro, Name: "Savings vault", Kind: model.LedgerAccountAsset},
		openingBalances: &model.LedgerAccount{ID: 200, CurrencyID: 1, Currency: euro, Name: "Opening balances", Kind: model.LedgerAccountEquity},
		fxClearing:      &model.LedgerAccount{ID: 201, CurrencyID: 1, Currency: euro, Name: "FX clearing", Kind: model.LedgerAccountAsset},
	}
}

// mockLookups mocks the currency, financial account, and institution lookups.
func (f addBalanceFixtures) mockLookups(env *testsuite.TestWorkflowEnvironment) {
	env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, &ct.RetrieveCurrencyParams{Code: "EUR"}).Return(f.euro, nil)
	env.OnWorkflow(ledgeraccountswf.RetrieveFinancialAccount, mock.Anything, lt.RetrieveFinancialAccountParams{ID: f.account.ID}).
		Return(&lt.RetrieveFinancialAccountResult{FinancialAccount: f.account}, nil)
	env.OnWorkflow(institutionwf.RetrieveInstitution, mock.Anything, it.RetrieveInstitutionParams{ID: f.institution.ID}).
		Return(&it.RetrieveInstitutionResult{Institution: f.institution}, nil)
}

// mockLedgerAccounts mocks creating and retrieving the vault and the shared
// accounts, as the child workflows would find or create them.
func (f addBalanceFixtures) mockLedgerAccounts(env *testsuite.TestWorkflowEnvironment) {
	for input, account := range map[lt.CreateLedgerAccountInput]*model.LedgerAccount{
		{FinancialAccountID: nil, CurrencyID: 1, Name: "Opening balances", Kind: model.LedgerAccountEquity}: f.openingBalances,
		{FinancialAccountID: nil, CurrencyID: 1, Name: "FX clearing", Kind: model.LedgerAccountAsset}:       f.fxClearing,
	} {
		env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, input).Return(&lt.CreateLedgerAccountResult{LedgerAccountID: account.ID}, nil)
	}
	env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, lt.CreateLedgerAccountInput{
		FinancialAccountID: ptr.Int64(f.account.ID), CurrencyID: 1, Name: "Savings vault", Kind: model.LedgerAccountAsset,
	}).Return(&lt.CreateLedgerAccountResult{LedgerAccountID: f.vault.ID}, nil)
	for _, account := range []*model.LedgerAccount{f.vault, f.openingBalances, f.fxClearing} {
		env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: account.ID}).
			Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: account}, nil)
	}
}

func (f addBalanceFixtures) wantBalances() []lt.AccountSummary {
	return []lt.AccountSummary{
		{
			LedgerAccountID: 100, LedgerAccountName: "Savings vault", Kind: model.LedgerAccountAsset, CurrencyID: 1, CurrencyCode: "EUR",
			FinancialAccountID: ptr.Int64(10), FinancialAccountName: "Revolut Personal", InstitutionName: "Revolut",
		},
		{LedgerAccountID: 200, LedgerAccountName: "Opening balances", Kind: model.LedgerAccountEquity, CurrencyID: 1, CurrencyCode: "EUR"},
		{LedgerAccountID: 201, LedgerAccountName: "FX clearing", Kind: model.LedgerAccountAsset, CurrencyID: 1, CurrencyCode: "EUR"},
	}
}

// openingEntry matches the journal entry want by what it books rather than by
// its representation: a decimal such as 500.50 arrives as 500.5, equal in
// value but not in its fields, and no metadata arrives as an empty map.
func openingEntry(want lt.PostJournalEntryInput) any {
	return mock.MatchedBy(func(got lt.PostJournalEntryInput) bool {
		if got.IdempotencyKey != want.IdempotencyKey || !got.OccurredAt.Equal(want.OccurredAt) ||
			*got.Description != *want.Description || len(got.Metadata) != 0 || len(got.Postings) != len(want.Postings) {
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

// TestAddBalance checks that the balance is created under the financial
// account with the shared accounts of its currency, that its opening balance
// is booked against "Opening balances" with a key derived from the balance,
// at the given instant or else at the time of the request, and that no entry
// is posted without an opening balance.
func (s *WorkflowUnitTestSuite) TestAddBalance() {
	f := newAddBalanceFixtures()
	start := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	given := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		input ast.AddBalanceInput
		// postedAt is when the opening entry is expected, none if zero.
		postedAt time.Time
	}{
		"with an opening balance at a given instant": {
			input:    ast.AddBalanceInput{FinancialAccountID: 10, Name: "Savings vault", CurrencyCode: "EUR", OpeningBalance: decimal.RequireFromString("500.50"), OpeningBalanceAt: given},
			postedAt: given,
		},
		"with an opening balance at the time of the request": {
			input:    ast.AddBalanceInput{FinancialAccountID: 10, Name: "Savings vault", CurrencyCode: "EUR", OpeningBalance: decimal.RequireFromString("500.50")},
			postedAt: start,
		},
		"without an opening balance": {
			input: ast.AddBalanceInput{FinancialAccountID: 10, Name: "Savings vault", CurrencyCode: "EUR"},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			env.SetStartTime(start)
			f.mockLookups(env)
			f.mockLedgerAccounts(env)
			want := &ast.AddBalanceResult{Outcome: ast.OutcomeOpened, LedgerAccountID: 100, Balances: ast.Balances{Balances: f.wantBalances()}}
			if !tc.postedAt.IsZero() {
				env.OnWorkflow(journalwf.PostJournalEntry, mock.Anything, openingEntry(lt.PostJournalEntryInput{
					IdempotencyKey: "opening-balance:100",
					OccurredAt:     tc.postedAt,
					Description:    ptr.NewPtr("Opening balance"),
					Postings: []lt.PostingInput{
						{LedgerAccountID: 100, Amount: decimal.RequireFromString("500.50"), Side: model.PostingDebit},
						{LedgerAccountID: 200, Amount: decimal.RequireFromString("500.50"), Side: model.PostingCredit},
					},
				})).Return(&lt.PostJournalEntryResult{JournalEntryID: 500}, nil).Once()
				want.OpeningEntries = []ast.OpeningEntry{{LedgerAccountID: 100, JournalEntryID: 500}}
			}

			env.ExecuteWorkflow(AddBalance, tc.input)

			require.NoError(s.T(), env.GetWorkflowError())
			var got *ast.AddBalanceResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), want, got)
			env.AssertExpectations(s.T())
		})
	}
}

// TestAddBalance_Errors checks that an invalid request, an unknown currency, or
// a closed account or institution fails before any ledger account is created,
// since none of those child workflows is mocked, and that a failing step's
// error is returned.
func (s *WorkflowUnitTestSuite) TestAddBalance_Errors() {
	f := newAddBalanceFixtures()
	closedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	input := ast.AddBalanceInput{FinancialAccountID: 10, Name: "Savings vault", CurrencyCode: "EUR"}

	tests := map[string]struct {
		input     ast.AddBalanceInput
		mock      func(env *testsuite.TestWorkflowEnvironment)
		wantError string
	}{
		"an invalid request": {
			input:     ast.AddBalanceInput{FinancialAccountID: 10, CurrencyCode: "EUR"},
			mock:      func(*testsuite.TestWorkflowEnvironment) {},
			wantError: "validation error",
		},
		"an unknown currency": {
			input: input,
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, mock.Anything).Return(nil, errors.New("resource Currency not found"))
			},
			wantError: "retrieve currency EUR",
		},
		"a closed financial account": {
			input: input,
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, mock.Anything).Return(f.euro, nil)
				closed := *f.account
				closed.ClosedAt = &closedAt
				env.OnWorkflow(ledgeraccountswf.RetrieveFinancialAccount, mock.Anything, mock.Anything).
					Return(&lt.RetrieveFinancialAccountResult{FinancialAccount: &closed}, nil)
			},
			wantError: "resource FinancialAccount with ID: 10 is closed",
		},
		"a closed institution": {
			input: input,
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(currencieswf.RetrieveCurrency, mock.Anything, mock.Anything).Return(f.euro, nil)
				env.OnWorkflow(ledgeraccountswf.RetrieveFinancialAccount, mock.Anything, mock.Anything).
					Return(&lt.RetrieveFinancialAccountResult{FinancialAccount: f.account}, nil)
				closed := *f.institution
				closed.ClosedAt = &closedAt
				env.OnWorkflow(institutionwf.RetrieveInstitution, mock.Anything, mock.Anything).
					Return(&it.RetrieveInstitutionResult{Institution: &closed}, nil)
			},
			wantError: "resource Institution with ID: 2 is closed",
		},
		"creating the balance fails": {
			input: input,
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				f.mockLookups(env)
				env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, mock.Anything).Return(nil, errors.New("database down"))
			},
			wantError: "create ledger account Savings vault",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			tc.mock(env)
			env.ExecuteWorkflow(AddBalance, tc.input)

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
