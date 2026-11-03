package categories

// CategorySummary is a flat projection of model.LedgerAccount. The model
// itself cannot be used as an MCP tool's output type directly: its
// Parent/Children/FinancialAccount relation fields form a cycle, and
// jsonschema-go's reflection-based schema inference panics on a cyclic Go
// type rather than emitting a recursive schema.
type CategorySummary struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	ParentID   *int64 `json:"parentId,omitempty"`
	CurrencyID int64  `json:"currencyId"`
	Kind       string `json:"kind"`
	Closed     bool   `json:"closed"`
}

// ListCategoriesToolResult is list_categories's output.
type ListCategoriesToolResult struct {
	Categories []CategorySummary `json:"categories"`
}
