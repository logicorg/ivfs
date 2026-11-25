package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_RenameCategory() {
	tests := []struct {
		name string
		// category is the fixture's name for the category to rename.
		category string
		newName  string
		// wantName is the name stored, when it is not the one given.
		wantName string
		wantErr  string
	}{
		{name: "another name", category: "travel", newName: "Trips"},
		{name: "a name with spaces around it", category: "travel", newName: "  Trips ", wantName: "Trips"},
		{name: "its own name in another casing", category: "travel", newName: "TRAVEL"},
		{name: "the name of a category under another parent", category: "fruit", newName: "Kitchen"},
		{name: "the name of a category of the other kind", category: "travel", newName: "Jobs"},
		{name: "the name of a top-level category in another casing", category: "travel", newName: "FOOD", wantErr: "Category conflict"},
		{name: "the name of a closed sibling", category: "travel", newName: "Old", wantErr: "Category conflict"},
		{name: "the name of the entries without a category", category: "travel", newName: "Uncategorized", wantErr: "needs another name"},
		{name: "no name", category: "travel", newName: " ", wantErr: "Name"},
		{name: "a category that does not exist", category: "missing", newName: "Trips", wantErr: "Category not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := s.seed()
			defer s.clear()
			id := *f.id(tt.category)

			activityResult, err := s.Env.ExecuteActivity(s.ca.RenameCategory, ct.RenameCategoryInput{CategoryID: id, Name: tt.newName})

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				if category := s.stored(id); category != nil {
					require.Equal(s.T(), f[tt.category].Name, category.Name)
				}
				return
			}
			require.NoError(s.T(), err)
			var result ct.CategoryResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.Equal(s.T(), id, result.CategoryID)

			wantName := tt.newName
			if tt.wantName != "" {
				wantName = tt.wantName
			}
			renamed := s.stored(id)
			require.Equal(s.T(), wantName, renamed.Name)
			require.True(s.T(), renamed.UpdatedAt.After(f[tt.category].UpdatedAt))
		})
	}
}

func (s *UnitTestSuite) Test_RenameCategory_IsIdempotent() {
	f := s.seed()
	input := ct.RenameCategoryInput{CategoryID: f["travel"].ID, Name: "Trips"}

	for range 2 {
		activityResult, err := s.Env.ExecuteActivity(s.ca.RenameCategory, input)
		require.NoError(s.T(), err)
		var result ct.CategoryResult
		require.NoError(s.T(), activityResult.Get(&result))
		require.Equal(s.T(), input.CategoryID, result.CategoryID)
		require.Equal(s.T(), "Trips", s.stored(input.CategoryID).Name)
	}
}
