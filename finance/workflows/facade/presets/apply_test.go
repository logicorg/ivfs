package presets

import (
	"fmt"
	"testing"

	cut "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	pt "github.com/logicorg/ivfs/finance/common/types/presettypes"
	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	tagswf "github.com/logicorg/ivfs/finance/workflows/tags"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type WorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestWorkflowUnitTestSuite(t *testing.T) { suite.Run(t, new(WorkflowUnitTestSuite)) }

// recorder stands in for the three child workflows: it gives each item the
// next id, in the order the items are created, and logs every call, so a test
// sees both what was created and in which order. failOn names the one item
// whose creation fails.
type recorder struct {
	nextID int64
	calls  []string
	failOn string
}

func (r *recorder) mock(env *testsuite.TestWorkflowEnvironment) {
	env.OnWorkflow(currencieswf.InsertCurrency, mock.Anything, mock.Anything).Return(
		func(_ workflow.Context, p ct.InsertCurrencyParams) (ct.InsertCurrencyResult, error) {
			id, err := r.create("currency " + p.Code)
			return ct.InsertCurrencyResult{CurrencyID: id}, err
		})
	env.OnWorkflow(tagswf.InsertTag, mock.Anything, mock.Anything).Return(
		func(_ workflow.Context, p tt.InsertTagParams) (*tt.InsertTagResult, error) {
			id, err := r.create("tag " + p.Name)
			if err != nil {
				return nil, err
			}
			return &tt.InsertTagResult{ID: id}, nil
		})
	env.OnWorkflow(categorieswf.CreateCategory, mock.Anything, mock.Anything).Return(
		func(_ workflow.Context, in cut.CreateCategoryInput) (*cut.CategoryResult, error) {
			parent := "none"
			if in.ParentID != nil {
				parent = fmt.Sprint(*in.ParentID)
			}
			id, err := r.create(fmt.Sprintf("category %s %s under %s", in.Kind, in.Name, parent))
			if err != nil {
				return nil, err
			}
			return &cut.CategoryResult{CategoryID: id}, nil
		})
}

func (r *recorder) create(call string) (int64, error) {
	r.calls = append(r.calls, call)
	if call == r.failOn {
		return 0, internalerrors.ErrResourceConflict{ResourceType: "Currency", Reason: "the name is taken"}
	}
	r.nextID++
	return r.nextID, nil
}

func (s *WorkflowUnitTestSuite) TestApplyPreset() {
	tests := map[string]struct {
		input     pt.ApplyPresetInput
		wantCalls []string
		want      pt.ApplyPresetResult
	}{
		"currencies, then tags, then categories parent first": {
			input: pt.ApplyPresetInput{
				Currencies: []ct.InsertCurrencyParams{{Code: "EUR", Name: "Euro"}, {Code: "USD", Name: "US Dollar"}},
				Tags:       []tt.InsertTagParams{{Name: "vacation"}},
				Categories: []pt.PresetCategory{
					{Name: "Food", Kind: model.LedgerAccountExpense, Children: []pt.PresetCategory{
						{Name: "Groceries", Children: []pt.PresetCategory{{Name: "Organic"}}},
						{Name: "Eating out"},
					}},
					{Name: "Salary", Kind: model.LedgerAccountIncome},
				},
			},
			wantCalls: []string{
				"currency EUR",
				"currency USD",
				"tag vacation",
				"category expense Food under none",
				"category expense Groceries under 4",
				"category expense Organic under 5",
				"category expense Eating out under 4",
				"category income Salary under none",
			},
			want: pt.ApplyPresetResult{
				CurrencyIDs: []int64{1, 2},
				TagIDs:      []int64{3},
				Categories: []pt.PresetCategoryResult{
					{CategoryID: 4, Children: []pt.PresetCategoryResult{
						{CategoryID: 5, Children: []pt.PresetCategoryResult{{CategoryID: 6}}},
						{CategoryID: 7},
					}},
					{CategoryID: 8},
				},
			},
		},
		"only tags": {
			input:     pt.ApplyPresetInput{Tags: []tt.InsertTagParams{{Name: "vacation"}, {Name: "recurring"}}},
			wantCalls: []string{"tag vacation", "tag recurring"},
			want:      pt.ApplyPresetResult{CurrencyIDs: []int64{}, TagIDs: []int64{1, 2}, Categories: []pt.PresetCategoryResult{}},
		},
		"the default preset": {
			input: pt.Default(),
			want:  defaultResult(),
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			r := &recorder{}
			r.mock(env)

			env.ExecuteWorkflow(ApplyPreset, tc.input)

			require.NoError(s.T(), env.GetWorkflowError())
			var got pt.ApplyPresetResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), tc.want, got)
			if tc.wantCalls != nil {
				require.Equal(s.T(), tc.wantCalls, r.calls)
			}
		})
	}
}

// defaultResult is the result of the default preset with ids given in order:
// the 5 currencies, the 4 tags, then each category before its children.
func defaultResult() pt.ApplyPresetResult {
	preset := pt.Default()
	id := int64(len(preset.Currencies) + len(preset.Tags))
	var tree func([]pt.PresetCategory) []pt.PresetCategoryResult
	tree = func(categories []pt.PresetCategory) []pt.PresetCategoryResult {
		var out []pt.PresetCategoryResult
		for _, c := range categories {
			id++
			parent := pt.PresetCategoryResult{CategoryID: id}
			parent.Children = tree(c.Children)
			out = append(out, parent)
		}
		return out
	}
	return pt.ApplyPresetResult{
		CurrencyIDs: []int64{1, 2, 3, 4, 5},
		TagIDs:      []int64{6, 7, 8, 9},
		Categories:  tree(preset.Categories),
	}
}

// TestApplyPreset_Errors checks that an invalid preset writes nothing, and
// that a failure stops the workflow where it happens, keeping what was
// created before it.
func (s *WorkflowUnitTestSuite) TestApplyPreset_Errors() {
	input := pt.ApplyPresetInput{
		Currencies: []ct.InsertCurrencyParams{{Code: "EUR", Name: "Euro"}, {Code: "EUX", Name: "Euro two"}},
		Tags:       []tt.InsertTagParams{{Name: "vacation"}},
		Categories: []pt.PresetCategory{
			{Name: "Food", Kind: model.LedgerAccountExpense, Children: []pt.PresetCategory{{Name: "Groceries"}, {Name: "Eating out"}}},
		},
	}

	tests := map[string]struct {
		input     pt.ApplyPresetInput
		failOn    string
		wantCalls []string
		wantError string
	}{
		"an invalid preset": {
			input:     pt.ApplyPresetInput{Categories: []pt.PresetCategory{{Name: "Food"}}},
			wantError: `category "Food" needs a kind`,
		},
		"a currency fails": {
			input:     input,
			failOn:    "currency EUX",
			wantCalls: []string{"currency EUR", "currency EUX"},
			wantError: "insert currency EUX",
		},
		"a tag fails": {
			input:     input,
			failOn:    "tag vacation",
			wantCalls: []string{"currency EUR", "currency EUX", "tag vacation"},
			wantError: "insert tag vacation",
		},
		"a child category fails": {
			input:  input,
			failOn: "category expense Groceries under 4",
			wantCalls: []string{
				"currency EUR", "currency EUX", "tag vacation",
				"category expense Food under none", "category expense Groceries under 4",
			},
			wantError: "create category Groceries",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			r := &recorder{failOn: tc.failOn}
			r.mock(env)

			env.ExecuteWorkflow(ApplyPreset, tc.input)

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
			require.Equal(s.T(), tc.wantCalls, r.calls)
		})
	}
}
