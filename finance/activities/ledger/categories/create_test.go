package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_CreateCategory() {
	expense, income := model.LedgerAccountExpense, model.LedgerAccountIncome

	tests := []struct {
		name string
		// category is the name given, and parent the fixture's name for the
		// parent, or "" for the top level.
		category string
		kind     model.LedgerAccountKind
		parent   string
		// wantName is the name stored, when it is not the one given.
		wantName string
		// wantExisting is the fixture's name for the category returned in
		// place of a new one.
		wantExisting string
		wantErr      string
	}{
		{name: "a top-level category", category: "Health", kind: expense},
		{name: "an income category", category: "Dividends", kind: income},
		{name: "a child", category: "Vegetables", kind: expense, parent: "food"},
		{name: "a grandchild", category: "Berries", kind: expense, parent: "fruit"},
		{name: "a name with spaces around it", category: "  Health  ", kind: expense, wantName: "Health"},
		{name: "the name of a category under another parent", category: "Kitchen", kind: expense, parent: "food"},
		{name: "the name of a category of the other kind", category: "Travel", kind: income},
		{name: "the name of a top-level category in another casing", category: "TRAVEL", kind: expense, wantExisting: "travel"},
		{name: "the name of a sibling in another casing", category: " fruit ", kind: expense, parent: "food", wantExisting: "fruit"},
		{name: "the name of a closed category", category: "Old", kind: expense, wantExisting: "old"},
		{name: "a fourth level", category: "Lemons", kind: expense, parent: "citrus", wantErr: "nest at most 2 levels"},
		{name: "under a closed parent", category: "Older", kind: expense, parent: "old", wantErr: "is closed"},
		{name: "under a parent of the other kind", category: "Bonus", kind: expense, parent: "jobs", wantErr: "is of kind income"},
		{name: "under a parent with entries", category: "Deposit", kind: expense, parent: "rent", wantErr: "has entries"},
		{name: "under a parent that does not exist", category: "Orphan", kind: expense, parent: "missing", wantErr: "Category not found"},
		{name: "the name of the entries without a category", category: " uncategorized ", kind: expense, wantErr: "needs another name"},
		{name: "no name", category: "  ", kind: expense, wantErr: "Name"},
		{name: "a kind that is not income or expense", category: "Cash", kind: model.LedgerAccountAsset, wantErr: `must be "income" or "expense"`},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := s.seed()
			defer s.clear()
			before := s.Count[model.Category]()

			activityResult, err := s.Env.ExecuteActivity(s.ca.CreateCategory, ct.CreateCategoryInput{Name: tt.category, Kind: tt.kind, ParentID: f.id(tt.parent)})

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				require.Equal(s.T(), before, s.Count[model.Category]())
				return
			}
			require.NoError(s.T(), err)
			var result ct.CategoryResult
			require.NoError(s.T(), activityResult.Get(&result))

			if tt.wantExisting != "" {
				existing := f[tt.wantExisting]
				require.Equal(s.T(), existing.ID, result.CategoryID)
				require.Equal(s.T(), before, s.Count[model.Category]())
				require.Equal(s.T(), existing.Name, s.stored(existing.ID).Name, "the category keeps its casing")
				return
			}
			require.Equal(s.T(), before+1, s.Count[model.Category]())

			wantName := tt.category
			if tt.wantName != "" {
				wantName = tt.wantName
			}
			created := s.stored(result.CategoryID)
			require.Equal(s.T(), wantName, created.Name)
			require.Equal(s.T(), tt.kind, created.Kind)
			require.Equal(s.T(), f.id(tt.parent), created.ParentID)
			require.Nil(s.T(), created.ClosedAt)
		})
	}
}

func (s *UnitTestSuite) Test_CreateCategory_IsIdempotent() {
	input := ct.CreateCategoryInput{Name: "Health", Kind: model.LedgerAccountExpense}

	var ids []int64
	for range 2 {
		activityResult, err := s.Env.ExecuteActivity(s.ca.CreateCategory, input)
		require.NoError(s.T(), err)
		var result ct.CategoryResult
		require.NoError(s.T(), activityResult.Get(&result))
		ids = append(ids, result.CategoryID)
	}

	require.Equal(s.T(), ids[0], ids[1])
	require.Equal(s.T(), 1, s.Count[model.Category]())
}
