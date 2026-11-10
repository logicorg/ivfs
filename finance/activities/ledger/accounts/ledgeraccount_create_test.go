package accounts

import (
	"fmt"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/stretchr/testify/require"
)

// ledgerAccountFixtures is the reference data the ledger account creation
// tests build on. Cases read it and never change it.
type ledgerAccountFixtures struct {
	euro, dollar       *model.Currency
	personal, business *model.FinancialAccount
	food, household    *model.LedgerAccount
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
	f.food = &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Food", Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, f.food)
	f.household = &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Household", Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, f.household)
	return f
}

func (s *UnitTestSuite) createLedgerAccount(input lt.CreateLedgerAccountInput) lt.CreateLedgerAccountResult {
	s.T().Helper()

	activityResult, err := s.Env.ExecuteActivity(s.aa.CreateLedgerAccount, input)
	require.NoError(s.T(), err)
	var result lt.CreateLedgerAccountResult
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
// and the name; elsewhere it is the parent, the kind, the currency, and the
// name. The existing account is returned unchanged, closed or not, whatever
// the rest of the request says.
func (s *UnitTestSuite) Test_CreateLedgerAccount_RetrievesExisting() {
	f := s.seedLedgerAccountFixtures()
	closedAt := time.Now().UTC().Truncate(time.Second)

	tests := map[string]struct {
		existing func() *model.LedgerAccount
		input    lt.CreateLedgerAccountInput
	}{
		"under a financial account, in another casing": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset}
			},
			input: lt.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: " cash ", Kind: model.LedgerAccountAsset},
		},
		"under a financial account, of another kind": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset}
			},
			input: lt.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountLiability},
		},
		"a top-level category in another casing": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Groceries", Kind: model.LedgerAccountExpense}
			},
			input: lt.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "GROCERIES", Kind: model.LedgerAccountExpense},
		},
		"a child category in another casing": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{ParentID: &f.food.ID, CurrencyID: f.euro.ID, Name: "Supermarket", Kind: model.LedgerAccountExpense}
			},
			input: lt.CreateLedgerAccountInput{ParentID: &f.food.ID, CurrencyID: f.euro.ID, Name: "supermarket", Kind: model.LedgerAccountExpense},
		},
		"a closed account": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Savings vault", Kind: model.LedgerAccountAsset, ClosedAt: &closedAt}
			},
			input: lt.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Savings vault", Kind: model.LedgerAccountAsset},
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
// another financial account or parent, of another kind at the top level, or
// at the top level rather than under a financial account, is a new account.
func (s *UnitTestSuite) Test_CreateLedgerAccount_Scope() {
	f := s.seedLedgerAccountFixtures()
	cashUnderPersonal := func() *model.LedgerAccount {
		return &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset}
	}
	interestIncome := func() *model.LedgerAccount {
		return &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Interest", Kind: model.LedgerAccountIncome}
	}

	tests := map[string]struct {
		existing func() *model.LedgerAccount
		input    lt.CreateLedgerAccountInput
	}{
		"the same name in another currency, under a financial account": {
			existing: cashUnderPersonal,
			input:    lt.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.dollar.ID, Name: "Cash", Kind: model.LedgerAccountAsset},
		},
		"the same name under another financial account": {
			existing: cashUnderPersonal,
			input:    lt.CreateLedgerAccountInput{FinancialAccountID: &f.business.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset},
		},
		"the same name at the top level, rather than under a financial account": {
			existing: cashUnderPersonal,
			input:    lt.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset},
		},
		"the same top-level name of another kind": {
			existing: interestIncome,
			input:    lt.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "Interest", Kind: model.LedgerAccountExpense},
		},
		"the same top-level name in another currency": {
			existing: interestIncome,
			input:    lt.CreateLedgerAccountInput{CurrencyID: f.dollar.ID, Name: "Interest", Kind: model.LedgerAccountIncome},
		},
		"the same child name under another parent": {
			existing: func() *model.LedgerAccount {
				return &model.LedgerAccount{ParentID: &f.food.ID, CurrencyID: f.euro.ID, Name: "Supermarket", Kind: model.LedgerAccountExpense}
			},
			input: lt.CreateLedgerAccountInput{ParentID: &f.household.ID, CurrencyID: f.euro.ID, Name: "Supermarket", Kind: model.LedgerAccountExpense},
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

type ledgerAccountOutcome struct {
	result *lt.CreateLedgerAccountResult
	err    error
}

// Test_CreateLedgerAccount_RetryWaitsForConcurrentWriter checks that a create
// racing another one for the same key, with the name in another casing,
// returns the account the other one wrote instead of creating a second: the
// lookup cannot see the uncommitted row, so it is the unique index on the key
// that stops the second insert until the first commits. At the top level the
// parent is missing on both, which the index has to treat as the same value.
func (s *UnitTestSuite) Test_CreateLedgerAccount_RetryWaitsForConcurrentWriter() {
	f := s.seedLedgerAccountFixtures()

	tests := map[string]struct {
		first *model.LedgerAccount
		input lt.CreateLedgerAccountInput
	}{
		"a top-level category": {
			first: &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Groceries", Kind: model.LedgerAccountExpense},
			input: lt.CreateLedgerAccountInput{CurrencyID: f.euro.ID, Name: "groceries", Kind: model.LedgerAccountExpense},
		},
		"an account under a financial account": {
			first: &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset},
			input: lt.CreateLedgerAccountInput{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: " cash ", Kind: model.LedgerAccountAsset},
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
// be created under a closed financial account or a closed parent, and that the
// failure is ErrResourceClosed, which workflows treat as non-retryable, rather
// than a conflict another request could avoid. Nothing is inserted.
func (s *UnitTestSuite) Test_CreateLedgerAccount_ClosedScope() {
	f := s.seedLedgerAccountFixtures()
	closedAt := time.Now().UTC().Truncate(time.Second)

	tests := map[string]struct {
		// scope inserts the closed financial account or parent, and returns
		// the input that creates an account under it.
		scope     func() (lt.CreateLedgerAccountInput, func())
		wantError func(input lt.CreateLedgerAccountInput) string
	}{
		"under a closed financial account": {
			scope: func() (lt.CreateLedgerAccountInput, func()) {
				closed := &model.FinancialAccount{InstitutionID: f.personal.InstitutionID, Name: "Revolut Old", ClosedAt: &closedAt}
				temporaltesting.Insert(s.T(), s.DB, closed)
				input := lt.CreateLedgerAccountInput{FinancialAccountID: &closed.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset}
				return input, func() { temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.FinancialAccount)(nil), closed.ID) }
			},
			wantError: func(input lt.CreateLedgerAccountInput) string {
				return fmt.Sprintf("resource FinancialAccount with ID: %d is closed", *input.FinancialAccountID)
			},
		},
		"under a closed parent": {
			scope: func() (lt.CreateLedgerAccountInput, func()) {
				closed := &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Pets", Kind: model.LedgerAccountExpense, ClosedAt: &closedAt}
				temporaltesting.Insert(s.T(), s.DB, closed)
				input := lt.CreateLedgerAccountInput{ParentID: &closed.ID, CurrencyID: f.euro.ID, Name: "Vet", Kind: model.LedgerAccountExpense}
				return input, func() { s.deleteLedgerAccounts(closed.ID) }
			},
			wantError: func(input lt.CreateLedgerAccountInput) string {
				return fmt.Sprintf("parent ledger account %d is closed", *input.ParentID)
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
