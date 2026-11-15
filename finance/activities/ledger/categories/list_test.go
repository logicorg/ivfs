package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_ListCategories() {
	tests := []struct {
		name string
		// parent is the fixture's name for the parent to filter by.
		parent        string
		kind          model.LedgerAccountKind
		nameContains  string
		includeClosed bool
		// want is the names of the categories listed, in order.
		want    []string
		wantErr string
	}{
		{
			name: "every open category, by name",
			want: []string{"Attic", "Bills", "Citrus", "Food", "Fruit", "Home", "Jobs", "Kitchen", "Kitchen", "Power", "Rent", "Travel", "Wages"},
		},
		{
			name: "the closed ones too", includeClosed: true,
			want: []string{"Attic", "Bills", "Citrus", "Dusty", "Food", "Fruit", "Home", "Jobs", "Kitchen", "Kitchen", "Old", "Power", "Rent", "Travel", "Wages"},
		},
		{name: "one kind", kind: model.LedgerAccountIncome, want: []string{"Jobs", "Wages"}},
		{name: "the children of a parent", parent: "food", want: []string{"Fruit"}},
		{name: "the closed children of a parent", parent: "attic", includeClosed: true, want: []string{"Dusty"}},
		{name: "a parent whose children are closed", parent: "attic", want: []string{}},
		{name: "a part of the name, in another casing", nameContains: "ITCH", want: []string{"Kitchen", "Kitchen"}},
		{name: "a part of no name", nameContains: "zzz", want: []string{}},
		{name: "a kind that is not income or expense", kind: model.LedgerAccountAsset, wantErr: `must be "income" or "expense"`},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := s.seed()
			defer s.clear()

			activityResult, err := s.Env.ExecuteActivity(s.ca.ListCategories, ct.ListCategoriesParams{
				ParentID: f.id(tt.parent), Kind: tt.kind, NameContains: tt.nameContains, IncludeClosed: tt.includeClosed,
			})

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				return
			}
			require.NoError(s.T(), err)
			var result ct.ListCategoriesResult
			require.NoError(s.T(), activityResult.Get(&result))

			names := make([]string, 0, len(result.Categories))
			for _, category := range result.Categories {
				names = append(names, category.Name)
			}
			require.Equal(s.T(), tt.want, names)
		})
	}
}
