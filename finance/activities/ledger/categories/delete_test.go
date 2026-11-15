package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_DeleteCategory() {
	tests := []struct {
		name string
		// category and replacement are the fixture's names for the category to
		// delete and the one to merge it into, which is "" for none.
		category    string
		replacement string
		wantErr     string
	}{
		{name: "a category without entries", category: "travel"},
		{name: "a category with entries, which lose it", category: "rent"},
		{name: "a category with entries, which take the replacement", category: "rent", replacement: "travel"},
		{name: "a category without entries into a replacement", category: "travel", replacement: "rent"},
		{name: "a child", category: "power"},
		{name: "a child into a child of another parent", category: "power", replacement: "kitchen"},
		{name: "a closed category", category: "old"},
		{name: "a category that does not exist", category: "missing"},
		{name: "a category that does not exist into a replacement", category: "missing", replacement: "travel"},
		{name: "a category with children", category: "food", wantErr: "has categories under it"},
		{name: "into a closed replacement", category: "rent", replacement: "old", wantErr: "is closed"},
		{name: "into a replacement of the other kind", category: "rent", replacement: "jobs", wantErr: "is of kind income"},
		{name: "into a replacement with children", category: "rent", replacement: "food", wantErr: "has categories under it"},
		{name: "into a replacement that does not exist", category: "rent", replacement: "missing", wantErr: "does not exist"},
		{name: "into itself", category: "rent", replacement: "rent", wantErr: "cannot be merged into itself"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := s.seed()
			defer s.clear()
			id, replacementID := *f.id(tt.category), f.id(tt.replacement)
			categories, postings, entries := s.Count[model.Category](), s.Count[model.Posting](), s.Count[model.JournalEntry]()
			carried := s.CountWhere[model.Posting]("category_id = ?", id)
			replaced := 0
			if replacementID != nil {
				replaced = s.CountWhere[model.Posting]("category_id = ?", *replacementID)
			}

			_, err := s.Env.ExecuteActivity(s.ca.DeleteCategory, ct.DeleteCategoryInput{CategoryID: id, ReplacementID: replacementID})

			require.Equal(s.T(), postings, s.Count[model.Posting](), "no posting is deleted")
			require.Equal(s.T(), entries, s.Count[model.JournalEntry](), "no entry is deleted")
			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				require.Equal(s.T(), categories, s.Count[model.Category]())
				require.Equal(s.T(), carried, s.CountWhere[model.Posting]("category_id = ?", id))
				return
			}
			require.NoError(s.T(), err)

			require.Nil(s.T(), s.stored(id))
			require.Zero(s.T(), s.CountWhere[model.Posting]("category_id = ?", id))
			if tt.category == "missing" {
				require.Equal(s.T(), categories, s.Count[model.Category]())
				return
			}
			require.Equal(s.T(), categories-1, s.Count[model.Category]())
			if replacementID != nil {
				require.Equal(s.T(), replaced+carried, s.CountWhere[model.Posting]("category_id = ?", *replacementID))
			}
		})
	}
}

func (s *UnitTestSuite) Test_DeleteCategory_IsIdempotent() {
	f := s.seed()
	input := ct.DeleteCategoryInput{CategoryID: f["rent"].ID, ReplacementID: &f["travel"].ID}

	for range 2 {
		_, err := s.Env.ExecuteActivity(s.ca.DeleteCategory, input)
		require.NoError(s.T(), err)
		require.Nil(s.T(), s.stored(input.CategoryID))
		require.Equal(s.T(), 3, s.CountWhere[model.Posting]("category_id = ?", *input.ReplacementID))
	}
}
