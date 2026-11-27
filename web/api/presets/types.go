package presets

import (
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	pt "github.com/logicorg/ivfs/finance/common/types/presettypes"
	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/pkg/model"
)

// Preset is a set of currencies, tags, and categories to create: what the
// default preset holds, and what a client sends back once the user edited it.
type Preset struct {
	Currencies []ct.InsertCurrencyParams `json:"currencies" maxItems:"100"`
	Tags       []PresetTag               `json:"tags" maxItems:"100"`
	Categories []PresetCategory          `json:"categories" doc:"at most 100, counting children"`
}

// PresetTag is a tag of a preset.
type PresetTag struct {
	Name        string  `json:"name" minLength:"1"`
	Description *string `json:"description,omitempty"`
}

// PresetCategory is a category of a preset, with its children.
type PresetCategory struct {
	Name     string           `json:"name" minLength:"1"`
	Kind     string           `json:"kind,omitempty" enum:"income,expense" doc:"required on a top-level category, left out on a child, which takes its parent's"`
	Children []PresetCategory `json:"children,omitempty" doc:"at most two levels below a top-level category"`
}

// DefaultPresetInput is the request of GET /api/presets/default.
type DefaultPresetInput struct{}

// DefaultPresetOutput is the response of GET /api/presets/default.
type DefaultPresetOutput struct {
	Body Preset
}

// ApplyPresetInput is the request of POST /api/presets/apply.
type ApplyPresetInput struct {
	Body Preset
}

// AppliedPreset holds the id of every item created or found, in the order
// and shape of the request.
type AppliedPreset struct {
	CurrencyIDs []int64           `json:"currencyIds"`
	TagIDs      []int64           `json:"tagIds"`
	Categories  []AppliedCategory `json:"categories"`
}

// AppliedCategory is the id of a category and of its children.
type AppliedCategory struct {
	ID       int64             `json:"id"`
	Children []AppliedCategory `json:"children,omitempty"`
}

// ApplyPresetOutput is the response of POST /api/presets/apply.
type ApplyPresetOutput struct {
	Body AppliedPreset
}

// newPreset is a preset as the API returns it.
func newPreset(preset pt.ApplyPresetInput) Preset {
	out := Preset{
		Currencies: preset.Currencies,
		Tags:       make([]PresetTag, 0, len(preset.Tags)),
		Categories: newCategories(preset.Categories),
	}
	for _, tag := range preset.Tags {
		out.Tags = append(out.Tags, PresetTag{Name: tag.Name, Description: tag.Description})
	}
	return out
}

func newCategories(categories []pt.PresetCategory) []PresetCategory {
	out := make([]PresetCategory, 0, len(categories))
	for _, category := range categories {
		c := PresetCategory{Name: category.Name, Kind: string(category.Kind)}
		if len(category.Children) > 0 {
			c.Children = newCategories(category.Children)
		}
		out = append(out, c)
	}
	return out
}

// applyPresetInput is preset as ApplyPreset takes it.
func applyPresetInput(preset Preset) pt.ApplyPresetInput {
	out := pt.ApplyPresetInput{
		Currencies: preset.Currencies,
		Tags:       make([]tt.InsertTagParams, 0, len(preset.Tags)),
		Categories: presetCategories(preset.Categories),
	}
	for _, tag := range preset.Tags {
		out.Tags = append(out.Tags, tt.InsertTagParams{Name: tag.Name, Description: tag.Description})
	}
	return out
}

func presetCategories(categories []PresetCategory) []pt.PresetCategory {
	out := make([]pt.PresetCategory, 0, len(categories))
	for _, category := range categories {
		c := pt.PresetCategory{Name: category.Name, Kind: model.LedgerAccountKind(category.Kind)}
		if len(category.Children) > 0 {
			c.Children = presetCategories(category.Children)
		}
		out = append(out, c)
	}
	return out
}

func newAppliedCategories(categories []pt.PresetCategoryResult) []AppliedCategory {
	out := make([]AppliedCategory, 0, len(categories))
	for _, category := range categories {
		c := AppliedCategory{ID: category.CategoryID}
		if len(category.Children) > 0 {
			c.Children = newAppliedCategories(category.Children)
		}
		out = append(out, c)
	}
	return out
}
