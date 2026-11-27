package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_PreviewCategoryDeletion() {
	tests := []struct {
		name string
		// category is the fixture's name for the category to preview.
		category string
		// id is used in place of category when it is not zero.
		id          int64
		wantEntries int
		wantErr     string
	}{
		{name: "a category without entries", category: "travel"},
		{name: "a category with one entry", category: "power", wantEntries: 1},
		{name: "an entry with two postings in the category counts once", category: "rent", wantEntries: 2},
		{name: "an income category", category: "wages", wantEntries: 1},
		{name: "a parent, whose children's entries are not its own", category: "bills"},
		{name: "a category that does not exist", category: "missing", wantErr: "Category not found"},
		{name: "an id that is not one", id: -1, wantErr: "validation error"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := s.seed()
			defer s.clear()
			id := tt.id
			if id == 0 {
				id = *f.id(tt.category)
			}
			categories, carried := s.Count[model.Category](), s.CountWhere[model.Posting]("category_id IS NOT NULL")

			activityResult, err := s.Env.ExecuteActivity(s.ca.PreviewCategoryDeletion, ct.PreviewCategoryDeletionInput{CategoryID: id})

			require.Equal(s.T(), categories, s.Count[model.Category](), "a preview changes nothing")
			require.Equal(s.T(), carried, s.CountWhere[model.Posting]("category_id IS NOT NULL"), "a preview changes nothing")
			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				return
			}
			require.NoError(s.T(), err)
			var result ct.CategoryDeletionPreview
			require.NoError(s.T(), activityResult.Get(&result))
			require.Equal(s.T(), ct.CategoryDeletionPreview{Entries: tt.wantEntries}, result)
		})
	}
}
