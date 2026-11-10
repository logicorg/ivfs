package accounts

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) createFinancialAccount(input lt.CreateFinancialAccountInput) lt.CreateFinancialAccountResult {
	s.T().Helper()

	activityResult, err := s.Env.ExecuteActivity(s.aa.CreateFinancialAccount, input)
	require.NoError(s.T(), err)
	var result lt.CreateFinancialAccountResult
	require.NoError(s.T(), activityResult.Get(&result))
	return result
}

// deleteFinancialAccount deletes a financial account a case created, together
// with the ledger accounts created with it.
func (s *UnitTestSuite) deleteFinancialAccount(id int64) {
	s.T().Helper()

	temporaltesting.DeleteWhere(s.T(), s.DB, (*model.LedgerAccount)(nil), "financial_account_id = ?", id)
	temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.FinancialAccount)(nil), id)
}

// Test_CreateFinancialAccount checks that a financial account is stored with
// its name trimmed but in the casing it was given, and that the cash ledger
// account requested with it is called "Cash": its financial account and
// currency already identify it (ADR 0005).
func (s *UnitTestSuite) Test_CreateFinancialAccount() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	tests := map[string]struct {
		name     string
		cash     *int64
		stored   string
		wantCash bool
	}{
		"with a cash account":           {name: "Revolut Personal", cash: ptr.Int64(euro.ID), stored: "Revolut Personal", wantCash: true},
		"a name with surrounding space": {name: "  Revolut Joint  ", stored: "Revolut Joint"},
		"without a cash account":        {name: "Revolut Business", stored: "Revolut Business"},
	}

	for name, tc := range tests {
		var created int64

		s.Run(name, func() {
			result := s.createFinancialAccount(lt.CreateFinancialAccountInput{InstitutionID: institution.ID, Name: tc.name, CashCurrencyID: tc.cash})
			created = result.FinancialAccountID

			var account model.FinancialAccount
			temporaltesting.Retrieve(s.T(), s.DB, &account, result.FinancialAccountID)
			require.Equal(s.T(), tc.stored, account.Name)
			require.Equal(s.T(), institution.ID, account.InstitutionID)

			if !tc.wantCash {
				require.Nil(s.T(), result.CashLedgerAccountID)
				return
			}
			require.NotNil(s.T(), result.CashLedgerAccountID)
			var cash model.LedgerAccount
			temporaltesting.Retrieve(s.T(), s.DB, &cash, *result.CashLedgerAccountID)
			require.Equal(s.T(), "Cash", cash.Name)
			require.Equal(s.T(), euro.ID, cash.CurrencyID)
			require.Equal(s.T(), model.LedgerAccountAsset, cash.Kind)
			require.Equal(s.T(), result.FinancialAccountID, *cash.FinancialAccountID)
		})

		s.deleteFinancialAccount(created)
	}
}

// Test_CreateFinancialAccount_Invalid checks that input the account cannot be
// created from is rejected as a validation error before anything is written:
// no institution, a blank name, or a cash currency that is not positive.
func (s *UnitTestSuite) Test_CreateFinancialAccount_Invalid() {
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	tests := map[string]struct {
		input  lt.CreateFinancialAccountInput
		errStr string
	}{
		"no institution": {
			input:  lt.CreateFinancialAccountInput{Name: "Revolut Personal"},
			errStr: "validation error: institutionId and name are required",
		},
		"a blank name": {
			input:  lt.CreateFinancialAccountInput{InstitutionID: institution.ID, Name: "   "},
			errStr: "validation error: institutionId and name are required",
		},
		"a cash currency that is not positive": {
			input:  lt.CreateFinancialAccountInput{InstitutionID: institution.ID, Name: "Revolut Personal", CashCurrencyID: ptr.Int64(0)},
			errStr: "validation error: cashCurrencyId must be positive",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.CreateFinancialAccount, tc.input)

			require.ErrorContains(s.T(), err, tc.errStr)
			require.Zero(s.T(), s.Count[model.FinancialAccount](), "a rejected account must not be written")
		})

		_, err := s.DB.NewDelete().Model((*model.FinancialAccount)(nil)).Where("institution_id = ?", institution.ID).Exec(s.T().Context())
		require.NoError(s.T(), err)
	}
}

// Test_CreateFinancialAccount_Idempotent checks that a financial account is
// retrieved by its institution and its name, compared case-insensitively
// (ADR 0005): a retry, or a create of the same name in another casing,
// returns the first account and its cash ledger account, and changes nothing,
// whatever the rest of the request says. A retry asking for a cash account in
// another currency does not create one: the existing account is returned
// unchanged, and it has no cash account in that currency.
func (s *UnitTestSuite) Test_CreateFinancialAccount_Idempotent() {
	euro, dollar := testhelpers.Euro(), testhelpers.USDollar()
	temporaltesting.Insert(s.T(), s.DB, euro)
	temporaltesting.Insert(s.T(), s.DB, dollar)
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	input := lt.CreateFinancialAccountInput{InstitutionID: institution.ID, Name: "Revolut Personal", CashCurrencyID: ptr.Int64(euro.ID)}

	created := s.createFinancialAccount(input)
	retried := s.createFinancialAccount(input)

	require.Equal(s.T(), created, retried)
	require.Equal(s.T(), 1, s.Count[model.FinancialAccount]())
	require.Equal(s.T(), 1, s.Count[model.LedgerAccount]())

	different := s.createFinancialAccount(lt.CreateFinancialAccountInput{
		InstitutionID: institution.ID, Name: " revolut personal ", ExternalID: ptr.NewPtr("IBAN-1"), CashCurrencyID: ptr.Int64(euro.ID),
	})

	require.Equal(s.T(), created, different)
	var stored model.FinancialAccount
	temporaltesting.Retrieve(s.T(), s.DB, &stored, created.FinancialAccountID)
	require.Equal(s.T(), "Revolut Personal", stored.Name)
	require.Nil(s.T(), stored.ExternalID)

	otherCurrency := s.createFinancialAccount(lt.CreateFinancialAccountInput{
		InstitutionID: institution.ID, Name: "Revolut Personal", CashCurrencyID: ptr.Int64(dollar.ID),
	})

	require.Equal(s.T(), created.FinancialAccountID, otherCurrency.FinancialAccountID)
	require.Nil(s.T(), otherCurrency.CashLedgerAccountID)
	require.Equal(s.T(), 1, s.Count[model.FinancialAccount]())
	require.Equal(s.T(), 1, s.Count[model.LedgerAccount]())
}

// Test_CreateFinancialAccount_Scope checks that a name and an external ID are
// unique within their institution only: another institution may reuse them,
// while an external ID that another account of the same institution holds is
// a conflict, since it identifies a real account.
func (s *UnitTestSuite) Test_CreateFinancialAccount_Scope() {
	revolut := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, revolut)
	n26 := &model.Institution{Name: "N26", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, n26)

	tests := map[string]struct {
		input  lt.CreateFinancialAccountInput
		errStr string
	}{
		"the same name at another institution": {
			input: lt.CreateFinancialAccountInput{InstitutionID: n26.ID, Name: "Personal"},
		},
		"the same external id at another institution": {
			input: lt.CreateFinancialAccountInput{InstitutionID: n26.ID, Name: "Main", ExternalID: ptr.NewPtr("IBAN-1")},
		},
		"the same external id at the same institution": {
			input:  lt.CreateFinancialAccountInput{InstitutionID: revolut.ID, Name: "Joint", ExternalID: ptr.NewPtr("IBAN-1")},
			errStr: "resource FinancialAccount conflict",
		},
	}

	for name, tc := range tests {
		existing := &model.FinancialAccount{InstitutionID: revolut.ID, Name: "Personal", ExternalID: ptr.NewPtr("IBAN-1")}
		temporaltesting.Insert(s.T(), s.DB, existing)
		var created int64

		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.CreateFinancialAccount, tc.input)

			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
				require.Equal(s.T(), 1, s.Count[model.FinancialAccount](), "a rejected account must not be written")
				return
			}
			require.NoError(s.T(), err)
			var result lt.CreateFinancialAccountResult
			require.NoError(s.T(), activityResult.Get(&result))
			created = result.FinancialAccountID
			require.NotEqual(s.T(), existing.ID, result.FinancialAccountID)
		})

		s.deleteFinancialAccount(created)
		s.deleteFinancialAccount(existing.ID)
	}
}

// Test_CreateFinancialAccount_ReturnsClosedAccount checks that a name is
// reserved even by a closed account: a create of that name returns the closed
// account, as retrieve-or-insert does for any existing one (ADR 0005).
func (s *UnitTestSuite) Test_CreateFinancialAccount_ReturnsClosedAccount() {
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	closedAt := time.Now().UTC().Truncate(time.Second)
	closed := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut Joint", ClosedAt: &closedAt}
	temporaltesting.Insert(s.T(), s.DB, closed)

	result := s.createFinancialAccount(lt.CreateFinancialAccountInput{InstitutionID: institution.ID, Name: "revolut joint"})

	require.Equal(s.T(), closed.ID, result.FinancialAccountID)
	require.Equal(s.T(), 1, s.Count[model.FinancialAccount]())
}

type financialAccountOutcome struct {
	result *lt.CreateFinancialAccountResult
	err    error
}

// Test_CreateFinancialAccount_RetryWaitsForConcurrentWriter checks that a
// create racing another one for the same name, in another casing, returns the
// account the other one wrote instead of creating a second: the lookup cannot
// see the uncommitted row, so it is the unique index on the name that stops
// the second insert until the first commits.
func (s *UnitTestSuite) Test_CreateFinancialAccount_RetryWaitsForConcurrentWriter() {
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	writer, err := s.DB.BeginTx(s.T().Context(), nil)
	require.NoError(s.T(), err)
	s.T().Cleanup(func() { _ = writer.Rollback() })
	first := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut Personal"}
	_, err = writer.NewInsert().Model(first).Returning("id").Exec(s.T().Context())
	require.NoError(s.T(), err)

	done := make(chan financialAccountOutcome, 1)
	go func() {
		result, err := s.aa.CreateFinancialAccount(s.T().Context(), lt.CreateFinancialAccountInput{InstitutionID: institution.ID, Name: "revolut personal"})
		done <- financialAccountOutcome{result, err}
	}()
	select {
	case <-done:
		s.T().Fatal("the create finished without waiting for the transaction writing the same name")
	case <-time.After(500 * time.Millisecond):
	}

	require.NoError(s.T(), writer.Commit())
	outcome := <-done
	require.NoError(s.T(), outcome.err)
	require.Equal(s.T(), first.ID, outcome.result.FinancialAccountID)
	require.Equal(s.T(), 1, s.Count[model.FinancialAccount]())
}

// Creating a financial account under an institution or with a cash currency
// that does not exist has to be a not found error, which Temporal does not
// retry, and it has to leave nothing behind.
func (s *UnitTestSuite) TestCreateFinancialAccount_MissingParents() {
	currency := &model.Currency{Code: "EUR", Name: "Euro", Symbol: "€", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	tests := map[string]struct {
		input        lt.CreateFinancialAccountInput
		expectedType string
	}{
		"unknown institution": {
			input:        lt.CreateFinancialAccountInput{InstitutionID: 999999, Name: "Revolut EUR"},
			expectedType: "Institution",
		},
		"unknown cash currency": {
			input:        lt.CreateFinancialAccountInput{InstitutionID: institution.ID, Name: "Revolut EUR", CashCurrencyID: ptr.Int64(999999)},
			expectedType: "Currency",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.CreateFinancialAccount, tc.input)

			var appErr *temporal.ApplicationError
			require.ErrorAs(s.T(), err, &appErr)
			require.Equal(s.T(), "ErrNotFound", appErr.Type())
			require.ErrorContains(s.T(), err, tc.expectedType)

			count, err := s.DB.NewSelect().Model((*model.FinancialAccount)(nil)).Count(s.T().Context())
			require.NoError(s.T(), err)
			require.Zero(s.T(), count, "a failed creation must leave nothing behind")
		})
	}
}
