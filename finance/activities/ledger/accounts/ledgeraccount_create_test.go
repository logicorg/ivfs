package accounts

import (
	"fmt"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/stretchr/testify/require"
	"time"
)

// ledgerAccountFixtures is the reference data the ledger account creation
// tests build on. Cases read it and never change it.
type ledgerAccountFixtures struct {
	euro, dollar       *model.Currency
	personal, business *model.FinancialAccount
}

func (s *UnitTestSuite) seedLedgerAccountFixtures() ledgerAccountFixtures {
	s.T().Helper()

	f := ledgerAccountFixtures{euro: testhelpers.Euro(), dollar: testhelpers.USDollar()}
	temporaltesting.Insert(s.T(), s.DB, f.euro)
	temporaltesting.Insert(s.T(), s.DB, f.dollar)
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	f.personal = &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut Personal"}
	temporaltesting.Insert(s.T(), s.DB, f.personal)
	f.business = &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut Business"}
	temporaltesting.Insert(s.T(), s.DB, f.business)
	return f
}

func (s *UnitTestSuite) createLedgerAccount(input act.CreateLedgerAccountInput) act.CreateLedgerAccountResult {
	s.T().Helper()

	activityResult, err := s.Env.ExecuteActivity(s.aa.CreateLedgerAccount, input)
	require.NoError(s.T(), err)
	var result act.CreateLedgerAccountResult
	require.NoError(s.T(), activityResult.Get(&result))
	return result
}

func (s *UnitTestSuite) deleteLedgerAccounts(ids ...int64) {
	s.T().Helper()

	for _, id := range ids {
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.LedgerAccount)(nil), id)
	}
}

// Test_CreateLedgerAccount_RetrievesExisting checks that a ledger account is
// retrieved by its key, with the name compared case-insensitively (ADR 0005):
// under a financial account the key is the financial account, the currency,
// and the name; for the shared accounts it is the kind, the currency, and the
// name. The existing account is returned unchanged, closed or not, whatever
// the rest of the request says.
func (s *UnitTestSuite) Test_CreateLedgerAccount_RetrievesExisting() {
	f := s.seedLedgerAccountFixtures()
	closedAt := time.Now().UTC().Truncate(time.Second)

	tests := map[string]struct {
		existing func() *model.LedgerAccount
		input    act.CreateLedgerAccountInput
	}{
		"under a financial account, in another casing": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset}
			},
			input: act.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: " cash ", Kind: model.LedgerAccountAsset},
		},
		"under a financial account, of another kind": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset}
			},
			input: act.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountLiability},
		},
		"a shared account in another casing": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Expenses", Kind: model.LedgerAccountExpense}
			},
			input: act.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "EXPENSES", Kind: model.LedgerAccountExpense},
		},
		"a closed account": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Savings vault", Kind: model.LedgerAccountAsset, ClosedAt: &closedAt}
			},
			input: act.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Savings vault", Kind: model.LedgerAccountAsset},
		},
	}

	for name, tc := range tests {
		existing := tc.existing()
		temporaltesting.Insert(s.T(), s.DB, existing)
		before := s.Count[model.LedgerAccount]()
		var created int64

		s.Run(name, func() {
			result := s.createLedgerAccount(tc.input)
			created = result.LedgerAccountID

			require.Equal(s.T(), existing.ID, result.LedgerAccountID)
			require.Equal(s.T(), before, s.Count[model.LedgerAccount]())
			var stored model.LedgerAccount
			temporaltesting.Retrieve(s.T(), s.DB, &stored, existing.ID)
			require.Equal(s.T(), existing.Name, stored.Name)
			require.Equal(s.T(), existing.Kind, stored.Kind)
		})

		s.deleteLedgerAccounts(created, existing.ID)
	}
}

// Test_CreateLedgerAccount_Scope checks that what is not part of a key does
// not make two accounts the same: the same name in another currency, under
// another financial account, of another kind among the shared accounts, or
// shared rather than under a financial account, is a new account.
func (s *UnitTestSuite) Test_CreateLedgerAccount_Scope() {
	f := s.seedLedgerAccountFixtures()
	cashUnderPersonal := func() *model.LedgerAccount {
		return &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset}
	}
	sharedEquity := func() *model.LedgerAccount {
		return &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Adjustments", Kind: model.LedgerAccountEquity}
	}

	tests := map[string]struct {
		existing func() *model.LedgerAccount
		input    act.CreateLedgerAccountInput
	}{
		"the same name in another currency, under a financial account": {
			existing: cashUnderPersonal,
			input:    act.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.dollar.ID, Name: "Cash", Kind: model.LedgerAccountAsset},
		},
		"the same name under another financial account": {
			existing: cashUnderPersonal,
			input:    act.CreateLedgerAccountInput{FinancialAccountID: &f.business.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset},
		},
		"the same name with no financial account, rather than under one": {
			existing: cashUnderPersonal,
			input:    act.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset},
		},
		"the same shared name of another kind": {
			existing: sharedEquity,
			input:    act.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "Adjustments", Kind: model.LedgerAccountAsset},
		},
		"the same shared name in another currency": {
			existing: sharedEquity,
			input:    act.CreateLedgerAccountInput{CurrencyID: f.dollar.ID, Name: "Adjustments", Kind: model.LedgerAccountEquity},
		},
		"the shared income account in another currency": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Income", Kind: model.LedgerAccountIncome}
			},
			input: act.CreateLedgerAccountInput{CurrencyID: f.dollar.ID, Name: "Income", Kind: model.LedgerAccountIncome},
		},
	}

	for name, tc := range tests {
		existing := tc.existing()
		temporaltesting.Insert(s.T(), s.DB, existing)
		var created int64

		s.Run(name, func() {
			result := s.createLedgerAccount(tc.input)
			created = result.LedgerAccountID

			require.NotEqual(s.T(), existing.ID, result.LedgerAccountID)
		})

		s.deleteLedgerAccounts(created, existing.ID)
	}
}

// Test_CreateLedgerAccount_OnlySharedCategoryAccounts checks that the shared
// "Expenses" and "Income" are the only expense and income accounts: any other
// name, or one of them under a financial account, is refused, and nothing is
// inserted.
func (s *UnitTestSuite) Test_CreateLedgerAccount_OnlySharedCategoryAccounts() {
	f := s.seedLedgerAccountFixtures()

	tests := map[string]struct {
		input   act.CreateLedgerAccountInput
		wantErr string
	}{
		"an expense account of another name": {
			input:   act.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "Groceries", Kind: model.LedgerAccountExpense},
			wantErr: `the only expense account is "Expenses", with no financial account; create a category for "Groceries" instead`,
		},
		"an income account of another name": {
			input:   act.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "Salary", Kind: model.LedgerAccountIncome},
			wantErr: `the only income account is "Income", with no financial account; create a category for "Salary" instead`,
		},
		"an income account named as the expense one": {
			input:   act.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "Expenses", Kind: model.LedgerAccountIncome},
			wantErr: `the only income account is "Income"`,
		},
		"the expense account under a financial account": {
			input:   act.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Expenses", Kind: model.LedgerAccountExpense},
			wantErr: `the only expense account is "Expenses", with no financial account`,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			before := s.Count[model.LedgerAccount]()

			_, err := s.Env.ExecuteActivity(s.aa.CreateLedgerAccount, tc.input)
			require.ErrorContains(s.T(), err, tc.wantErr)

			require.Equal(s.T(), before, s.Count[model.LedgerAccount]())
		})
	}
}

// Test_CreateLedgerAccount_ReservedNames checks that the name of a shared or
// stray account is taken among the accounts of no financial account whatever
// the kind, and nothing is inserted, while a financial account may have an
// account of that name.
func (s *UnitTestSuite) Test_CreateLedgerAccount_ReservedNames() {
	f := s.seedLedgerAccountFixtures()

	tests := map[string]struct {
		input   act.CreateLedgerAccountInput
		wantErr string
	}{
		"the clearing account's name as equity": {
			input:   act.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: " fx CLEARING ", Kind: model.LedgerAccountEquity},
			wantErr: `"fx CLEARING" is the name of the ledger's asset account and cannot name an equity account with no financial account; pick another name`,
		},
		"the income account's name as an asset": {
			input:   act.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "Income", Kind: model.LedgerAccountAsset},
			wantErr: `"Income" is the name of the ledger's income account`,
		},
		"the stray's name as a liability": {
			input:   act.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "Untracked accounts", Kind: model.LedgerAccountLiability},
			wantErr: `"Untracked accounts" is the name of the ledger's equity account`,
		},
		"the clearing account's name under a financial account": {
			input: act.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "FX clearing", Kind: model.LedgerAccountLiability},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			before := s.Count[model.LedgerAccount]()

			activityResult, err := s.Env.ExecuteActivity(s.aa.CreateLedgerAccount, tc.input)
			if tc.wantErr == "" {
				require.NoError(s.T(), err)
				var result act.CreateLedgerAccountResult
				require.NoError(s.T(), activityResult.Get(&result))
				s.deleteLedgerAccounts(result.LedgerAccountID)
				return
			}
			require.ErrorContains(s.T(), err, tc.wantErr)

			require.Equal(s.T(), before, s.Count[model.LedgerAccount]())
		})
	}
}

type ledgerAccountOutcome struct {
	result *act.CreateLedgerAccountResult
	err    error
}

// Test_CreateLedgerAccount_RetryWaitsForConcurrentWriter checks that a create
// racing another one for the same key, with the name in another casing,
// returns the account the other one wrote instead of creating a second: the
// lookup cannot see the uncommitted row, so it is the unique index on the key
// that stops the second insert until the first commits.
func (s *UnitTestSuite) Test_CreateLedgerAccount_RetryWaitsForConcurrentWriter() {
	f := s.seedLedgerAccountFixtures()

	tests := map[string]struct {
		first *model.LedgerAccount
		input act.CreateLedgerAccountInput
	}{
		"a shared account": {
			first: &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Expenses", Kind: model.LedgerAccountExpense},
			input: act.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "expenses", Kind: model.LedgerAccountExpense},
		},
		"an account under a financial account": {
			first: &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset},
			input: act.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: " cash ", Kind: model.LedgerAccountAsset},
		},
	}

	for name, tc := range tests {
		var created int64

		s.Run(name, func() {
			writer, err := s.DB.BeginTx(s.T().Context(), nil)
			require.NoError(s.T(), err)
			s.T().Cleanup(func() { _ = writer.Rollback() })
			_, err = writer.NewInsert().Model(tc.first).Returning("id").Exec(s.T().Context())
			require.NoError(s.T(), err)

			done := make(chan ledgerAccountOutcome, 1)
			go func() {
				result, err := s.aa.CreateLedgerAccount(s.T().Context(), tc.input)
				done <- ledgerAccountOutcome{result, err}
			}()
			select {
			case outcome := <-done:
				if outcome.result != nil {
					created = outcome.result.LedgerAccountID
				}
				s.T().Fatal("the create finished without waiting for the transaction writing the same key")
			case <-time.After(500 * time.Millisecond):
			}

			require.NoError(s.T(), writer.Commit())
			outcome := <-done
			require.NoError(s.T(), outcome.err)
			created = outcome.result.LedgerAccountID
			require.Equal(s.T(), tc.first.ID, outcome.result.LedgerAccountID)
		})

		s.deleteLedgerAccounts(created, tc.first.ID)
	}
}

// Test_CreateLedgerAccount_ClosedScope checks that a new ledger account cannot
// be created under a closed financial account, and that the failure is ErrResourceClosed, which workflows treat as non-retryable, rather
// than a conflict another request could avoid. Nothing is inserted.
func (s *UnitTestSuite) Test_CreateLedgerAccount_ClosedScope() {
	f := s.seedLedgerAccountFixtures()
	closedAt := time.Now().UTC().Truncate(time.Second)

	tests := map[string]struct {
		// scope inserts the closed financial account, and returns the input
		// that creates an account under it.
		scope     func() (act.CreateLedgerAccountInput, func())
		wantError func(input act.CreateLedgerAccountInput) string
	}{
		"under a closed financial account": {
			scope: func() (act.CreateLedgerAccountInput, func()) {
				closed := &model.FinancialAccount{InstitutionID: f.personal.InstitutionID, Name: "Revolut Old", ClosedAt: &closedAt}
				temporaltesting.Insert(s.T(), s.DB, closed)
				input := act.CreateLedgerAccountInput{FinancialAccountID: &closed.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset}
				return input, func() { temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.FinancialAccount)(nil), closed.ID) }
			},
			wantError: func(input act.CreateLedgerAccountInput) string {
				return fmt.Sprintf("resource FinancialAccount with ID: %d is closed", *input.FinancialAccountID)
			},
		},
	}

	for name, tc := range tests {
		input, deleteScope := tc.scope()

		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.CreateLedgerAccount, input)
			require.ErrorContains(s.T(), err, tc.wantError(input))

			require.Zero(s.T(), s.CountWhere[model.LedgerAccount]("name = ?", input.Name))
		})

		deleteScope()
	}
}
