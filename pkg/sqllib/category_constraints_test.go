package sqllib_test

import (
	"strconv"
	"strings"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

// Test_CategoryConstraints checks the rules of a category that the database
// enforces: its kind, and a name that is unique among the siblings of a kind,
// whatever its casing. Each case starts from the expense categories "Food" and
// "Groceries" under it, and runs one insert directly, as anything other than
// the activities would.
//
// The statements name the id of "Food" with :food.
func (s *UnitTestSuite) Test_CategoryConstraints() {
	const (
		kindCheck = "category_kind_check"
		nameKey   = "category_name_lower_key"
	)

	tests := []struct {
		name      string
		statement string
		wantErr   string
	}{
		{
			name:      "another name at the top level",
			statement: `INSERT INTO ledger.category (kind, name) VALUES ('expense', 'Transport')`,
		},
		{
			name:      "the name of a top-level category of the same kind",
			statement: `INSERT INTO ledger.category (kind, name) VALUES ('expense', 'Food')`,
			wantErr:   nameKey,
		},
		{
			name:      "the name of a top-level category in another casing",
			statement: `INSERT INTO ledger.category (kind, name) VALUES ('expense', 'FOOD')`,
			wantErr:   nameKey,
		},
		{
			name:      "the name of a top-level category of the other kind",
			statement: `INSERT INTO ledger.category (kind, name) VALUES ('income', 'Food')`,
		},
		{
			name:      "the name of a sibling",
			statement: `INSERT INTO ledger.category (parent_id, kind, name) VALUES (:food, 'expense', 'groceries')`,
			wantErr:   nameKey,
		},
		{
			name:      "the name of its parent",
			statement: `INSERT INTO ledger.category (parent_id, kind, name) VALUES (:food, 'expense', 'Food')`,
		},
		{
			name:      "the name of a category under another parent",
			statement: `INSERT INTO ledger.category (kind, name) VALUES ('expense', 'Groceries')`,
		},
		{
			name:      "a kind that is not income or expense",
			statement: `INSERT INTO ledger.category (kind, name) VALUES ('asset', 'Cash')`,
			wantErr:   kindCheck,
		},
		{
			name:      "a parent that does not exist",
			statement: `INSERT INTO ledger.category (parent_id, kind, name) VALUES (999999, 'expense', 'Orphan')`,
			wantErr:   "category_parent_id_fkey",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			food := &model.Category{Kind: model.LedgerAccountExpense, Name: "Food"}
			temporaltesting.Insert(s.T(), s.DB, food)
			groceries := &model.Category{ParentID: &food.ID, Kind: model.LedgerAccountExpense, Name: "Groceries"}
			temporaltesting.Insert(s.T(), s.DB, groceries)
			// Children go first: a parent that is still referenced cannot be deleted.
			defer func() {
				_, err := s.DB.ExecContext(s.T().Context(), `DELETE FROM ledger.category WHERE parent_id IS NOT NULL`)
				require.NoError(s.T(), err)
				_, err = s.DB.ExecContext(s.T().Context(), `DELETE FROM ledger.category`)
				require.NoError(s.T(), err)
			}()

			statement := strings.ReplaceAll(tt.statement, ":food", strconv.FormatInt(food.ID, 10))
			_, err := s.DB.ExecContext(s.T().Context(), statement)

			if tt.wantErr == "" {
				require.NoError(s.T(), err)
				return
			}
			require.ErrorContains(s.T(), err, tt.wantErr)
		})
	}
}
