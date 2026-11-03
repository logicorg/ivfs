// Package accounts is the web API's account endpoints: the financial
// accounts a person holds at institutions, and the balances of their ledger
// accounts.
package accounts

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"accounts"}

// Register adds every account endpoint to api.
func Register(api huma.API, c client.Client) {
	registerList(api, c)
	registerBalances(api, c)
	registerLedgerBalance(api, c)
	registerCloseInstitution(api, c)
	registerCloseFinancialAccount(api, c)
	registerCloseLedgerAccount(api, c)
	registerDeleteInstitution(api, c)
	registerDeleteFinancialAccount(api, c)
	registerDeleteLedgerAccount(api, c)
	registerPreviewInstitutionDeletion(api, c)
	registerPreviewFinancialAccountDeletion(api, c)
	registerPreviewLedgerAccountDeletion(api, c)
	registerRenameFinancialAccount(api, c)
	registerRenameLedgerAccount(api, c)
	registerOpen(api, c)
	registerAddBalance(api, c)
}
