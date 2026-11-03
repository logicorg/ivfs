package accounts

// FinancialAccountSummary is a flat projection of model.FinancialAccount.
// The model itself cannot be used as an MCP tool's output type directly:
// FinancialAccount.LedgerAccounts and LedgerAccount.FinancialAccount form a
// cycle, and jsonschema-go's reflection-based schema inference panics on a
// cyclic Go type rather than emitting a recursive schema.
type FinancialAccountSummary struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	InstitutionID   int64  `json:"institutionId"`
	InstitutionName string `json:"institutionName,omitempty"`
	Closed          bool   `json:"closed"`
}

// ListFinancialAccountsToolResult is list_financial_accounts's output.
type ListFinancialAccountsToolResult struct {
	FinancialAccounts []FinancialAccountSummary `json:"financialAccounts"`
}
