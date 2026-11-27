package categorytypes

import "github.com/logicorg/ivfs/pkg/model"

type RetrieveCategoryParams struct {
	ID int64 `json:"id"`
}

type ListCategoriesParams struct {
	ParentID *int64 `json:"parentId,omitempty" jsonschema:"only the categories directly under this one"`
	// Kind is income or expense, and every kind when empty.
	Kind model.LedgerAccountKind `json:"kind,omitempty" jsonschema:"income or expense; both if omitted"`
	// NameContains matches a case-insensitive substring of the category's
	// name, for example to resolve "groceries" to a category before a write
	// command.
	NameContains string `json:"nameContains,omitempty"`
	// IncludeClosed also lists the closed categories. A closed category still
	// takes postings; it is only not offered.
	IncludeClosed bool `json:"includeClosed,omitempty" jsonschema:"also list closed categories, which are kept for their history and not meant for new entries"`
}

type RetrieveCategoryResult struct {
	Category *model.Category `json:"category"`
}

type ListCategoriesResult struct {
	Categories []model.Category `json:"categories"`
}
