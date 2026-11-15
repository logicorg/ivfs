package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_MoveCategory() {
	tests := []struct {
		name string
		// category and parent are the fixture's names for the category to move
		// and its new parent, which is "" for the top level.
		category string
		parent   string
		wantErr  string
	}{
		{name: "a top-level category under another", category: "travel", parent: "food"},
		{name: "a top-level category under a child", category: "travel", parent: "fruit"},
		{name: "a child to the top level", category: "fruit", parent: ""},
		{name: "a grandchild under its grandparent", category: "citrus", parent: "food"},
		{name: "a parent under a top-level category", category: "home", parent: "food"},
		{name: "a top-level category to the top level", category: "travel", parent: ""},
		{name: "a child under the parent it has", category: "fruit", parent: "food"},
		{name: "a category with entries", category: "rent", parent: "food", wantErr: "cannot be moved"},
		{name: "a category with entries below it", category: "bills", parent: "food", wantErr: "cannot be moved"},
		{name: "under a parent with entries", category: "travel", parent: "rent", wantErr: "has entries"},
		{name: "under a closed parent", category: "travel", parent: "old", wantErr: "is closed"},
		{name: "under a parent of the other kind", category: "travel", parent: "jobs", wantErr: "is of kind income"},
		{name: "under a grandchild", category: "travel", parent: "citrus", wantErr: "nest at most 2 levels"},
		{name: "a parent under a child", category: "home", parent: "fruit", wantErr: "nest at most 2 levels"},
		{name: "under itself", category: "food", parent: "food", wantErr: "cannot go under itself"},
		{name: "under a category below it", category: "food", parent: "citrus", wantErr: "cannot go under itself"},
		{name: "to where a sibling has its name", category: "kitchen", parent: "", wantErr: "Category conflict"},
		{name: "a category that does not exist", category: "missing", parent: "food", wantErr: "Category not found"},
		{name: "under a parent that does not exist", category: "travel", parent: "missing", wantErr: "Category not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := s.seed()
			defer s.clear()
			id := *f.id(tt.category)

			activityResult, err := s.Env.ExecuteActivity(s.ca.MoveCategory, ct.MoveCategoryInput{CategoryID: id, ParentID: f.id(tt.parent)})

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				if category := s.stored(id); category != nil {
					require.Equal(s.T(), f[tt.category].ParentID, category.ParentID)
				}
				return
			}
			require.NoError(s.T(), err)
			var result ct.CategoryResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.Equal(s.T(), id, result.CategoryID)
			require.Equal(s.T(), f.id(tt.parent), s.stored(id).ParentID)
		})
	}
}

func (s *UnitTestSuite) Test_MoveCategory_IsIdempotent() {
	f := s.seed()
	input := ct.MoveCategoryInput{CategoryID: f["travel"].ID, ParentID: &f["food"].ID}

	for range 2 {
		activityResult, err := s.Env.ExecuteActivity(s.ca.MoveCategory, input)
		require.NoError(s.T(), err)
		var result ct.CategoryResult
		require.NoError(s.T(), activityResult.Get(&result))
		require.Equal(s.T(), input.CategoryID, result.CategoryID)
		require.Equal(s.T(), input.ParentID, s.stored(input.CategoryID).ParentID)
	}
}
