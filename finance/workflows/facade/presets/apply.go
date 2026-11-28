// Package presets is the facade that applies a preset of currencies, tags,
// and categories, composing the workflows that create each.
package presets

import (
	"fmt"

	cut "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	pt "github.com/logicorg/ivfs/finance/common/types/presettypes"
	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	tagswf "github.com/logicorg/ivfs/finance/workflows/tags"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// ApplyPreset creates every currency, tag, and category of the preset, or
// finds the one that exists, one at a time and each parent before its
// children. It is not atomic: a failure keeps what was created, and applying
// the same preset again finds those and creates the rest. A category that
// exists closed is found like any other, so a child under it fails with
// ErrResourceClosed until the user reopens or unticks it.
func ApplyPreset(ctx workflow.Context, input pt.ApplyPresetInput) (*pt.ApplyPresetResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	result := &pt.ApplyPresetResult{
		CurrencyIDs: make([]int64, 0, len(input.Currencies)),
		TagIDs:      make([]int64, 0, len(input.Tags)),
		Categories:  make([]pt.PresetCategoryResult, 0, len(input.Categories)),
	}
	for _, currency := range input.Currencies {
		var inserted ct.InsertCurrencyResult
		if err := workflow.ExecuteChildWorkflow(ctx, currencieswf.InsertCurrency, currency).Get(ctx, &inserted); err != nil {
			return nil, fmt.Errorf("insert currency %s: %w", currency.Code, err)
		}
		result.CurrencyIDs = append(result.CurrencyIDs, inserted.CurrencyID)
	}
	for _, tag := range input.Tags {
		var inserted tt.InsertTagResult
		if err := workflow.ExecuteChildWorkflow(ctx, tagswf.InsertTag, tag).Get(ctx, &inserted); err != nil {
			return nil, fmt.Errorf("insert tag %s: %w", tag.Name, err)
		}
		result.TagIDs = append(result.TagIDs, inserted.ID)
	}
	for _, category := range input.Categories {
		created, err := createCategory(ctx, category, category.Kind, nil)
		if err != nil {
			return nil, err
		}
		result.Categories = append(result.Categories, created)
	}
	return result, nil
}

// createCategory creates a category under parentID, then its children under
// it, all of the kind of the top-level category they descend from.
func createCategory(ctx workflow.Context, category pt.PresetCategory, kind model.LedgerAccountKind, parentID *int64) (pt.PresetCategoryResult, error) {
	var created cut.CategoryResult
	err := workflow.ExecuteChildWorkflow(ctx, categorieswf.CreateCategory, cut.CreateCategoryInput{
		Name: category.Name, Kind: kind, ParentID: parentID,
	}).Get(ctx, &created)
	if err != nil {
		return pt.PresetCategoryResult{}, fmt.Errorf("create category %s: %w", category.Name, err)
	}

	result := pt.PresetCategoryResult{CategoryID: created.CategoryID}
	for _, child := range category.Children {
		createdChild, err := createCategory(ctx, child, kind, &created.CategoryID)
		if err != nil {
			return pt.PresetCategoryResult{}, err
		}
		result.Children = append(result.Children, createdChild)
	}
	return result, nil
}
