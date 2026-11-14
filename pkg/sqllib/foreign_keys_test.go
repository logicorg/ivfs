package sqllib_test

import (
	"fmt"
	"testing"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite
}

func (s *UnitTestSuite) BeforeTest(_, _ string) { sqllib.InitDatabaseT(s.T(), s.DB) }
func (s *UnitTestSuite) AfterTest(_, _ string)  { s.DropAllTables() }

func Test_UnitTestSuite(t *testing.T) { suite.Run(t, new(UnitTestSuite)) }

// Test_ForeignKeysAreDeclared checks that every reference between the tables is
// a foreign key that restricts deletion, and that there are no others.
func (s *UnitTestSuite) Test_ForeignKeysAreDeclared() {
	expected := []string{
		"asset.asset_value.asset_id -> asset.asset",
		"asset.asset_value.currency_id -> currency.currency",
		"currency.conversion.from_currency_id -> currency.currency",
		"currency.conversion.to_currency_id -> currency.currency",
		"account.financial_account.institution_id -> institution.institution",
		"ledger.ledger_account.parent_id -> ledger.ledger_account",
		"ledger.ledger_account.financial_account_id -> account.financial_account",
		"ledger.ledger_account.currency_id -> currency.currency",
		"ledger.category.parent_id -> ledger.category",
		"ledger.journal_entry.counterparty_id -> ledger.counterparty",
		"ledger.posting.journal_entry_id -> ledger.journal_entry",
		"ledger.posting.ledger_account_id -> ledger.ledger_account",
		"ledger.posting.category_id -> ledger.category",
		"asset.position_movement.journal_entry_id -> ledger.journal_entry",
		// A second key on the entry and its date, listed here by its first column.
		"asset.position_movement.journal_entry_id -> ledger.journal_entry",
		"asset.position_movement.financial_account_id -> account.financial_account",
		"asset.position_movement.asset_id -> asset.asset",
		"asset.position_movement.cost_basis_currency_id -> currency.currency",
		"settings.user_settings.default_currency_id -> currency.currency",
	}

	var declared []struct {
		Child      string `bun:"child"`
		Column     string `bun:"col"`
		Parent     string `bun:"parent"`
		DeleteType string `bun:"deltype"`
	}
	err := s.DB.NewRaw(`SELECT c.conrelid::regclass::text AS child, a.attname AS col, c.confrelid::regclass::text AS parent, c.confdeltype::text AS deltype
		FROM pg_constraint c
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
		WHERE c.contype = 'f'`).Scan(s.T().Context(), &declared)
	require.NoError(s.T(), err)

	got := make([]string, 0, len(declared))
	for _, fk := range declared {
		got = append(got, fmt.Sprintf("%s.%s -> %s", fk.Child, fk.Column, fk.Parent))
		require.Equal(s.T(), "r", fk.DeleteType, "%s.%s must restrict deletion", fk.Child, fk.Column)
	}
	require.ElementsMatch(s.T(), expected, got)
}

func (s *UnitTestSuite) Test_ParentNotFound() {
	_, err := s.DB.NewInsert().Model(&model.FinancialAccount{InstitutionID: 999999, Name: "orphan"}).Exec(s.T().Context())
	require.Error(s.T(), err)
	require.True(s.T(), sqllib.IsForeignKeyViolation(err))

	// Activities wrap the error they get from the database, and it has to be
	// recognised through the wrapping.
	mapped := sqllib.ParentNotFound(fmt.Errorf("insert financial account: %w", err))

	var notFound internalerrors.ErrNotFound
	require.ErrorAs(s.T(), mapped, &notFound)
	require.Equal(s.T(), "Institution", notFound.ResourceType)
	require.Equal(s.T(), "999999", notFound.ID)
}

func (s *UnitTestSuite) Test_StillReferenced() {
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	temporaltesting.Insert(s.T(), s.DB, &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut EUR"})

	_, err := s.DB.NewDelete().Model((*model.Institution)(nil)).Where("id = ?", institution.ID).Exec(s.T().Context())
	require.Error(s.T(), err)

	mapped := sqllib.StillReferenced(fmt.Errorf("delete institution: %w", err), "Institution")

	var conflict internalerrors.ErrResourceConflict
	require.ErrorAs(s.T(), mapped, &conflict)
	require.Equal(s.T(), "Institution", conflict.ResourceType)
	require.Contains(s.T(), conflict.Reason, "account.financial_account.institution_id")
}

// Errors that are not a foreign key violation have to pass through untouched,
// so that callers keep handling them as before.
func (s *UnitTestSuite) Test_OtherErrorsAreNotMapped() {
	temporaltesting.Insert(s.T(), s.DB, testhelpers.Euro())
	_, err := s.DB.NewInsert().Model(testhelpers.Euro()).Exec(s.T().Context())
	require.Error(s.T(), err, "a duplicate code violates a unique constraint")

	require.False(s.T(), sqllib.IsForeignKeyViolation(err))
	require.NoError(s.T(), sqllib.ParentNotFound(err))
	require.NoError(s.T(), sqllib.StillReferenced(err, "Currency"))
	require.NoError(s.T(), sqllib.ParentNotFound(nil))
	require.NoError(s.T(), sqllib.StillReferenced(nil, "Currency"))
}
