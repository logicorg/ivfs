package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_RetrieveCategory() {
	tests := []struct {
		name string
		// category is the fixture's name for the category to retrieve.
		category string
		// id is used in place of category when it is not zero.
		id      int64
		wantErr string
	}{
		{name: "a top-level category", category: "food"},
		{name: "a child", category: "fruit"},
		{name: "a closed category", category: "old"},
		{name: "a category that does not exist", category: "missing", wantErr: "Category not found"},
		{name: "an id that is not one", id: -1, wantErr: "must be positive"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := s.seed()
			defer s.clear()
			id := tt.id
			if id == 0 {
				id = *f.id(tt.category)
			}

			activityResult, err := s.Env.ExecuteActivity(s.ca.RetrieveCategory, ct.RetrieveCategoryParams{ID: id})

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				return
			}
			require.NoError(s.T(), err)
			var result ct.RetrieveCategoryResult
			require.NoError(s.T(), activityResult.Get(&result))

			want := f[tt.category]
			require.Equal(s.T(), want.ID, result.Category.ID)
			require.Equal(s.T(), want.Name, result.Category.Name)
			require.Equal(s.T(), model.LedgerAccountExpense, result.Category.Kind)
			require.Equal(s.T(), want.ParentID, result.Category.ParentID)
			require.Equal(s.T(), want.ClosedAt == nil, result.Category.ClosedAt == nil)
		})
	}
}
