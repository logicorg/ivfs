package categories

import (
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// Category is an income or expense category as the API returns it.
type Category struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Kind         string `json:"kind" enum:"income,expense"`
	CurrencyID   int64  `json:"currencyId"`
	CurrencyCode string `json:"currencyCode"`
	ParentID     *int64 `json:"parentId,omitempty" doc:"the parent category, of the same kind and currency; none for a top-level one"`
	Closed       bool   `json:"closed"`
}

// CategoryList is every category asked for, in no particular order.
type CategoryList struct {
	Categories []Category `json:"categories"`
}

// ListCategoriesInput is the request of GET /api/categories.
type ListCategoriesInput struct {
	IncludeClosed bool `query:"includeClosed" doc:"also list closed categories"`
}

// ListCategoriesOutput is the response of GET /api/categories.
type ListCategoriesOutput struct {
	Body CategoryList
}

// NewCategory is a category to create.
type NewCategory struct {
	Name       string `json:"name" minLength:"1"`
	Kind       string `json:"kind" enum:"income,expense"`
	CurrencyID int64  `json:"currencyId" minimum:"1"`
	ParentID   *int64 `json:"parentId,omitempty" minimum:"1" doc:"the parent category, of the same kind and currency; top level if omitted"`
}

// CreateCategoryInput is the request of POST /api/categories.
type CreateCategoryInput struct {
	Body NewCategory
}

// CreateCategoryOutput is the response of POST /api/categories: the
// category's id, created now or before.
type CreateCategoryOutput struct {
	Body ct.CategoryResult
}

// CategoryName is a category's new name.
type CategoryName struct {
	Name string `json:"name" minLength:"1"`
}

// RenameCategoryInput is the request of POST /api/categories/{id}/rename.
type RenameCategoryInput struct {
	ID   int64 `path:"id" minimum:"1"`
	Body CategoryName
}

// CategoryParent is where to move a category.
type CategoryParent struct {
	ParentID *int64 `json:"parentId,omitempty" minimum:"1" doc:"the new parent, of the same kind and currency; top level if omitted"`
}

// MoveCategoryInput is the request of POST /api/categories/{id}/move.
type MoveCategoryInput struct {
	ID   int64 `path:"id" minimum:"1"`
	Body CategoryParent
}

// CloseCategoryInput is the request of POST /api/categories/{id}/close.
type CloseCategoryInput struct {
	ID int64 `path:"id" minimum:"1"`
}

// CategoryTotal is what was earned in an income category, or spent in an
// expense category, during a period.
type CategoryTotal struct {
	ID           int64          `json:"id" doc:"the category"`
	Kind         string         `json:"kind" enum:"income,expense"`
	CurrencyID   int64          `json:"currencyId"`
	CurrencyCode string         `json:"currencyCode"`
	Total        gateway.Amount `json:"total" doc:"earned for income, spent for an expense; negative when refunds outweigh the spending"`
}

// CategoryTotalList is the total of each category with a posted entry in the
// period. A category without one is left out.
type CategoryTotalList struct {
	Totals []CategoryTotal `json:"totals"`
}

// CategoryTotalsInput is the request of GET /api/categories/totals.
type CategoryTotalsInput struct {
	From time.Time `query:"from" required:"true" doc:"count entries that occurred at or after this instant"`
	To   time.Time `query:"to" required:"true" doc:"count entries that occurred before this instant"`
}

// CategoryTotalsOutput is the response of GET /api/categories/totals.
type CategoryTotalsOutput struct {
	Body CategoryTotalList
}

// DeleteCategoryInput is the request of DELETE /api/categories/{id}.
type DeleteCategoryInput struct {
	ID            int64 `path:"id" minimum:"1"`
	ReplacementID int64 `query:"replacementId" minimum:"1" doc:"the category to merge this one into; merging is not supported yet"`
}

// DeletionPreviewInput is the request of GET /api/categories/{id}/deletion-preview.
type DeletionPreviewInput struct {
	ID int64 `path:"id" minimum:"1"`
}
