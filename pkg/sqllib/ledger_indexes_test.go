package sqllib_test

import (
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/require"
)

// Test_LedgerIndexes checks that the indexes the journal list and the category
// reads rely on exist on the columns they page and filter by, and that installing the schema again, as
// every start of the worker does, neither fails nor duplicates them.
func (s *UnitTestSuite) Test_LedgerIndexes() {
	require.NoError(s.T(), sqllib.InitDatabase(s.T().Context(), s.DB))

	var indexes []struct {
		Name string `bun:"indexname"`
		Def  string `bun:"indexdef"`
	}
	err := s.DB.NewRaw(`SELECT indexname, indexdef FROM pg_indexes
		WHERE schemaname = 'ledger' AND indexname IN ('journal_entry_occurred_at_idx', 'posting_category_idx', 'posting_ledger_account_idx')
		ORDER BY indexname`).Scan(s.T().Context(), &indexes)
	require.NoError(s.T(), err)

	require.Len(s.T(), indexes, 3)
	require.Equal(s.T(), "journal_entry_occurred_at_idx", indexes[0].Name)
	require.Contains(s.T(), indexes[0].Def, "ON ledger.journal_entry USING btree (occurred_at, id)")
	require.Equal(s.T(), "posting_category_idx", indexes[1].Name)
	require.Contains(s.T(), indexes[1].Def, "ON ledger.posting USING btree (category_id)")
	require.Equal(s.T(), "posting_ledger_account_idx", indexes[2].Name)
	require.Contains(s.T(), indexes[2].Def, "ON ledger.posting USING btree (ledger_account_id)")
}
