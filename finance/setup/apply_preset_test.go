package setup

import (
	"time"

	cut "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	pt "github.com/logicorg/ivfs/finance/common/types/presettypes"
	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/finance/registration"
	"github.com/logicorg/ivfs/finance/workflows/facade/presets"
	"github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

// Test_ApplyPreset applies the default preset against the real workflows,
// activities, and database. It checks what the mocked tests cannot: that the
// child workflows compose, that a category's children land under it, that
// a currency which exists (EUR, inserted by the suite) is found rather than
// refused, and that applying the preset again finds everything and creates
// nothing.
func (s *UnitTestSuite) Test_ApplyPreset() {
	preset := pt.Default()

	var applied pt.ApplyPresetResult
	s.runWorkflow(presets.ApplyPreset, preset, &applied)

	require.Equal(s.T(), s.eur.ID, applied.CurrencyIDs[0])
	require.Equal(s.T(), 5, s.Count[model.Currency]())
	require.Equal(s.T(), 4, s.Count[model.Tag]())
	require.Equal(s.T(), 24, s.Count[model.Category]())

	food := applied.Categories[1]
	require.Equal(s.T(), 2, s.CountWhere[model.Category]("parent_id = ? AND kind = ?", food.CategoryID, model.LedgerAccountExpense))
	require.Equal(s.T(), 5, s.CountWhere[model.Category]("parent_id IS NULL AND kind = ?", model.LedgerAccountIncome))

	var reapplied pt.ApplyPresetResult
	s.runWorkflow(presets.ApplyPreset, preset, &reapplied)
	require.Equal(s.T(), applied, reapplied)
	require.Equal(s.T(), 5, s.Count[model.Currency]())
	require.Equal(s.T(), 4, s.Count[model.Tag]())
	require.Equal(s.T(), 24, s.Count[model.Category]())
}

// Test_ApplyPreset_Failures checks the failures a preset can meet only in the
// database, and that what was created before them stays.
func (s *UnitTestSuite) Test_ApplyPreset_Failures() {
	tests := map[string]struct {
		// closeFood creates "Food" and closes it before the preset runs.
		closeFood bool
		input     pt.ApplyPresetInput
		wantError string
		wantTags  int
	}{
		"a currency whose name is taken": {
			input: pt.ApplyPresetInput{
				Currencies: []ct.InsertCurrencyParams{{Code: "EUX", Name: "Euro"}},
				Tags:       []tt.InsertTagParams{{Name: "vacation"}},
			},
			wantError: "resource Currency conflict",
		},
		"a child under a closed category": {
			closeFood: true,
			input: pt.ApplyPresetInput{
				Tags: []tt.InsertTagParams{{Name: "vacation"}},
				Categories: []pt.PresetCategory{
					{Name: "Food", Kind: model.LedgerAccountExpense, Children: []pt.PresetCategory{{Name: "Groceries"}}},
				},
			},
			wantError: "closed",
			wantTags:  1,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			if tc.closeFood {
				var food cut.CategoryResult
				s.runWorkflow(categories.CreateCategory, cut.CreateCategoryInput{Name: "Food", Kind: model.LedgerAccountExpense}, &food)
				s.runWorkflow(categories.CloseCategory, cut.CloseCategoryInput{CategoryID: food.CategoryID, ClosedAt: time.Now()}, nil)
			}

			env := s.NewTestWorkflowEnvironment()
			registration.Register(env, tenancy.Static(s.DB))
			env.ExecuteWorkflow(presets.ApplyPreset, tc.input)

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
			require.Equal(s.T(), tc.wantTags, s.Count[model.Tag]())

			_, err := s.DB.NewDelete().Model((*model.Tag)(nil)).Where("TRUE").Exec(s.T().Context())
			require.NoError(s.T(), err)
			_, err = s.DB.NewDelete().Model((*model.Category)(nil)).Where("TRUE").Exec(s.T().Context())
			require.NoError(s.T(), err)
		})
	}
}
