package categorytypes

import "github.com/logicorg/ivfs/pkg/model"

type RetrieveCategoryParams struct {
	ID int64 `json:"id"`
}

type ListCategoriesParams struct {
	ParentID   *int64 `json:"parentId,omitempty"`
	CurrencyID *int64 `json:"currencyId,omitempty"`
	// NameContains matches a case-insensitive substring of the category's
	// name, for example to resolve "groceries" to a category before a write
	// command.
	NameContains  string `json:"nameContains,omitempty"`
	IncludeClosed bool   `json:"includeClosed,omitempty"`
}

type RetrieveCategoryResult struct {
	LedgerAccount *model.LedgerAccount `json:"ledgerAccount"`
}

type ListCategoriesResult struct {
	LedgerAccounts []model.LedgerAccount `json:"ledgerAccounts"`
}
