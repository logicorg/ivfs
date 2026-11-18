// Package presettypes holds the default preset of currencies, categories, and
// tags, and the input of the workflow that applies a preset.
package presettypes

import (
	"fmt"
	"strings"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// MaxItems bounds each list of a preset, and with it the history of the
// workflow that applies it. Categories count with their children.
const MaxItems = 100

// maxDepth is the most parent links a category may have.
const maxDepth = 2

// ApplyPresetInput is what a preset creates: three independent lists, the
// categories as a tree because a child needs its parent's id.
type ApplyPresetInput struct {
	Currencies []ct.InsertCurrencyParams `json:"currencies"`
	Tags       []tt.InsertTagParams      `json:"tags"`
	Categories []PresetCategory          `json:"categories"`
}

// PresetCategory is a category with its children. Only a top-level category
// has a kind; its children inherit it, as they must.
type PresetCategory struct {
	Name     string                  `json:"name"`
	Kind     model.LedgerAccountKind `json:"kind,omitempty"`
	Children []PresetCategory        `json:"children,omitempty"`
}

// ApplyPresetResult holds the id of every item created or found, in the order
// and shape of the input.
type ApplyPresetResult struct {
	CurrencyIDs []int64                `json:"currencyIds"`
	TagIDs      []int64                `json:"tagIds"`
	Categories  []PresetCategoryResult `json:"categories"`
}

// PresetCategoryResult is the id of a category and of its children.
type PresetCategoryResult struct {
	CategoryID int64                  `json:"categoryId"`
	Children   []PresetCategoryResult `json:"children,omitempty"`
}

// Validate checks the whole preset before anything is written, so that a
// mistake late in the form does not leave the early items created. An item
// given twice is refused rather than found twice, since it is almost always a
// typo in the form.
func (i *ApplyPresetInput) Validate() error {
	if len(i.Currencies)+len(i.Tags)+len(i.Categories) == 0 {
		return errs.ValidationError{Err: fmt.Errorf("the preset is empty: add a currency, a tag, or a category")}
	}
	if err := i.validateCurrencies(); err != nil {
		return err
	}
	if err := i.validateTags(); err != nil {
		return err
	}
	return i.validateCategories()
}

func (i *ApplyPresetInput) validateCurrencies() error {
	if len(i.Currencies) > MaxItems {
		return tooMany("currencies", len(i.Currencies))
	}
	codes := make(map[string]bool, len(i.Currencies))
	names := make(map[string]bool, len(i.Currencies))
	for _, currency := range i.Currencies {
		if err := currency.Validate(); err != nil {
			// The currency's own message does not say which row it is about.
			return errs.ValidationError{Err: fmt.Errorf("currency %q: %w", currency.Code, err)}
		}
		code, name := strings.ToUpper(strings.TrimSpace(currency.Code)), normalize(currency.Name)
		if codes[code] {
			return errs.ValidationError{Err: fmt.Errorf("currency %s is given more than once", code)}
		}
		if names[name] {
			return errs.ValidationError{Err: fmt.Errorf("currency name %q is given more than once", currency.Name)}
		}
		codes[code], names[name] = true, true
	}
	return nil
}

func (i *ApplyPresetInput) validateTags() error {
	if len(i.Tags) > MaxItems {
		return tooMany("tags", len(i.Tags))
	}
	seen := make(map[string]bool, len(i.Tags))
	for _, tag := range i.Tags {
		name := normalize(tag.Name)
		if name == "" {
			return errs.ErrMissingRequiredParameter{Parameters: []string{"tag Name"}}
		}
		if seen[name] {
			return errs.ValidationError{Err: fmt.Errorf("tag %q is given more than once", strings.TrimSpace(tag.Name))}
		}
		seen[name] = true
	}
	return nil
}

func (i *ApplyPresetInput) validateCategories() error {
	if n := countCategories(i.Categories); n > MaxItems {
		return tooMany("categories", n)
	}
	// Siblings are told apart by kind as well as name, as the unique index
	// does, so a top-level "Other" may be both an income and an expense.
	seen := make(map[string]bool, len(i.Categories))
	for _, category := range i.Categories {
		if !model.IsCategoryKind(category.Kind) {
			return errs.ValidationError{Err: fmt.Errorf("category %q needs a kind, income or expense, not %q", category.Name, category.Kind)}
		}
		key := string(category.Kind) + "/" + normalize(category.Name)
		if seen[key] {
			return errs.ValidationError{Err: fmt.Errorf("%s category %q is given more than once", category.Kind, strings.TrimSpace(category.Name))}
		}
		seen[key] = true
		if err := validateCategory(category, 0); err != nil {
			return err
		}
	}
	return nil
}

// validateCategory checks a category's name and its children, depth being
// the category's number of parent links.
func validateCategory(category PresetCategory, depth int) error {
	name := strings.TrimSpace(category.Name)
	if name == "" {
		return errs.ErrMissingRequiredParameter{Parameters: []string{"category Name"}}
	}
	if len(category.Children) > 0 && depth == maxDepth {
		return errs.ValidationError{Err: fmt.Errorf("category %q is %d levels deep and cannot have children (at most %d parent links)", name, depth, maxDepth)}
	}
	seen := make(map[string]bool, len(category.Children))
	for _, child := range category.Children {
		if child.Kind != "" {
			return errs.ValidationError{Err: fmt.Errorf("category %q takes the kind of %q: leave its kind out", child.Name, name)}
		}
		key := normalize(child.Name)
		if key != "" && seen[key] {
			return errs.ValidationError{Err: fmt.Errorf("category %q is given more than once under %q", strings.TrimSpace(child.Name), name)}
		}
		seen[key] = true
		if err := validateCategory(child, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func countCategories(categories []PresetCategory) int {
	n := len(categories)
	for _, category := range categories {
		n += countCategories(category.Children)
	}
	return n
}

// normalize compares names the way the database does: ignoring case and
// surrounding space.
func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func tooMany(list string, n int) error {
	return errs.ValidationError{Err: fmt.Errorf("%d %s is too many: a preset holds at most %d", n, list, MaxItems)}
}
