package categorytypes

import "github.com/logicorg/ivfs/pkg/model"

// CreateCategoryInput represents the input for creating a new category (ledger account).
type CreateCategoryInput struct {
	// Name is the name of the new category.
	Name string `json:"name"`
	// ParentID is the ID of the parent ledger account, if any.
	ParentID *int64 `json:"parentId,omitempty,omitzero" jsonschema:"the parent category, of the same kind and currency, if any"`
	// CurrencyID is the ID of the currency associated with the new category.
	CurrencyID int64 `json:"currencyId"`
	// Kind is the kind of the new category (ledger account).
	Kind model.LedgerAccountKind `json:"kind" jsonschema:"income or expense"`
}

// CategoryResult represents the result of creating a new category.
type CategoryResult struct {
	// LedgerAccountID is the ID of the newly created category ledger account.
	LedgerAccountID int64 `json:"ledgerAccountId"`
}
