// Package registration registers every activity and workflow of the finance
// domain, so the worker and the tests that run workflows against the real
// activities register the same set and cannot disagree about it.
package registration

import (
	"github.com/logicorg/ivfs/finance/activities/asset"
	"github.com/logicorg/ivfs/finance/activities/asset/value"
	"github.com/logicorg/ivfs/finance/activities/currency/conversion"
	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	"github.com/logicorg/ivfs/finance/activities/institution"
	ledgeraccounts "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	ledgerfx "github.com/logicorg/ivfs/finance/activities/ledger/fx"
	ledgerjournal "github.com/logicorg/ivfs/finance/activities/ledger/journal"
	ledgerportfolio "github.com/logicorg/ivfs/finance/activities/ledger/portfolio"
	ledgerreporting "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	"github.com/logicorg/ivfs/finance/activities/tag"
	usersettings "github.com/logicorg/ivfs/finance/activities/user/settings"
	"github.com/logicorg/ivfs/finance/workflows/assets"
	"github.com/logicorg/ivfs/finance/workflows/assets/assetvalues"
	"github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/finance/workflows/facade/accountsetup"
	"github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/finance/workflows/facade/investments"
	"github.com/logicorg/ivfs/finance/workflows/facade/transactions"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	ledgerfxwf "github.com/logicorg/ivfs/finance/workflows/ledger/fx"
	ledgerjournalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	ledgerportfoliowf "github.com/logicorg/ivfs/finance/workflows/ledger/portfolio"
	ledgerreportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	settingswf "github.com/logicorg/ivfs/finance/workflows/settings"
	"github.com/logicorg/ivfs/finance/workflows/tags"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"go.temporal.io/sdk/worker"
)

// Register registers every activity, opening its database through pools,
// and every workflow on r, which is a worker or a test workflow environment.
func Register(r worker.Registry, pools tenancy.Pools) {
	accountWrapperActivity := institution.Institution{Pools: pools}
	assetActivity := asset.AssetActivity{Pools: pools}
	assetValueActivity := value.AssetValueActivity{Pools: pools}
	currencyActivity := currency.Currency{Pools: pools}
	conversionActivity := conversion.Conversion{Pools: pools}
	tagActivity := tag.Tag{Pools: pools}
	ledgerAccountsActivity := ledgeraccounts.Activities{Pools: pools}
	ledgerFXActivity := ledgerfx.Activities{Pools: pools}
	ledgerJournalActivity := ledgerjournal.Activities{Pools: pools}
	ledgerPortfolioActivity := ledgerportfolio.Activities{Pools: pools}
	ledgerReportingActivity := ledgerreporting.Activities{Pools: pools}
	userSettingsActivity := usersettings.Activities{Pools: pools}

	r.RegisterActivity(&accountWrapperActivity)
	r.RegisterActivity(&assetActivity)
	r.RegisterActivity(&assetValueActivity)
	r.RegisterActivity(&currencyActivity)
	r.RegisterActivity(&conversionActivity)
	r.RegisterActivity(&tagActivity)
	r.RegisterActivity(&ledgerAccountsActivity)
	r.RegisterActivity(&ledgerFXActivity)
	r.RegisterActivity(&ledgerJournalActivity)
	r.RegisterActivity(&ledgerPortfolioActivity)
	r.RegisterActivity(&ledgerReportingActivity)
	r.RegisterActivity(&userSettingsActivity)

	r.RegisterWorkflow(assets.DeleteAsset)
	r.RegisterWorkflow(assets.InsertAsset)
	r.RegisterWorkflow(assets.RetrieveAllAssets)
	r.RegisterWorkflow(assets.RetrieveAsset)
	r.RegisterWorkflow(assets.ListAssets)

	r.RegisterWorkflow(assetvalues.RetrieveAssetValueHistory)

	r.RegisterWorkflow(accountsetup.AddBalance)
	r.RegisterWorkflow(accountsetup.OpenAccount)

	r.RegisterWorkflow(categories.CreateCategory)
	r.RegisterWorkflow(categories.RetrieveCategory)
	r.RegisterWorkflow(categories.ListCategories)
	r.RegisterWorkflow(categories.RenameCategory)
	r.RegisterWorkflow(categories.MoveCategory)
	r.RegisterWorkflow(categories.CloseCategory)
	r.RegisterWorkflow(categories.DeleteCategory)
	r.RegisterWorkflow(categories.PreviewCategoryDeletion)

	r.RegisterWorkflow(transactions.RecordExpense)
	r.RegisterWorkflow(transactions.RecordIncome)
	r.RegisterWorkflow(transactions.RecordTransfer)

	r.RegisterWorkflow(investments.RecordInvestment)

	r.RegisterWorkflow(currencies.DeleteCurrency)
	r.RegisterWorkflow(currencies.InsertCurrency)
	r.RegisterWorkflow(currencies.ListCurrencies)
	r.RegisterWorkflow(currencies.RetrieveConversionRateHistory)
	r.RegisterWorkflow(currencies.RetrieveCurrency)
	r.RegisterWorkflow(currencies.UpdateCurrency)

	r.RegisterWorkflow(tags.InsertTag)

	r.RegisterWorkflow(settingswf.RetrieveUserSettings)
	r.RegisterWorkflow(settingswf.UpdateUserSettings)

	r.RegisterWorkflow(institutionwf.InsertInstitution)
	r.RegisterWorkflow(institutionwf.RetrieveInstitution)
	r.RegisterWorkflow(institutionwf.UpdateInstitution)
	r.RegisterWorkflow(institutionwf.CloseInstitution)
	r.RegisterWorkflow(institutionwf.DeleteInstitution)

	r.RegisterWorkflow(ledgerfxwf.RegisterFXConversion)
	r.RegisterWorkflow(ledgerjournalwf.PostJournalEntry)
	r.RegisterWorkflow(ledgerjournalwf.RetrieveJournalEntry)
	r.RegisterWorkflow(ledgerjournalwf.ReverseJournalEntry)
	r.RegisterWorkflow(ledgerjournalwf.DeleteJournalEntry)
	r.RegisterWorkflow(ledgerjournalwf.ListJournalEntries)
	r.RegisterWorkflow(ledgerreportingwf.LedgerAccountBalance)
	r.RegisterWorkflow(ledgeraccountswf.CreateFinancialAccount)
	r.RegisterWorkflow(ledgeraccountswf.RetrieveFinancialAccount)
	r.RegisterWorkflow(ledgeraccountswf.ListFinancialAccounts)
	r.RegisterWorkflow(ledgeraccountswf.UpdateFinancialAccountMetadata)
	r.RegisterWorkflow(ledgeraccountswf.CloseFinancialAccount)
	r.RegisterWorkflow(ledgeraccountswf.CreateLedgerAccount)
	r.RegisterWorkflow(ledgeraccountswf.CreateCounterparty)
	r.RegisterWorkflow(ledgeraccountswf.RetrieveCounterparty)
	r.RegisterWorkflow(ledgeraccountswf.ListCounterparties)
	r.RegisterWorkflow(ledgeraccountswf.RenameCounterparty)
	r.RegisterWorkflow(ledgeraccountswf.DeleteCounterparty)
	r.RegisterWorkflow(ledgeraccountswf.RetrieveLedgerAccount)
	r.RegisterWorkflow(ledgeraccountswf.ListLedgerAccounts)
	r.RegisterWorkflow(ledgeraccountswf.ListAccounts)
	r.RegisterWorkflow(ledgeraccountswf.RenameLedgerAccount)
	r.RegisterWorkflow(ledgeraccountswf.MoveLedgerAccount)
	r.RegisterWorkflow(ledgeraccountswf.CloseLedgerAccount)
	r.RegisterWorkflow(ledgeraccountswf.DeleteLedgerAccount)
	r.RegisterWorkflow(ledgeraccountswf.DeleteFinancialAccount)
	r.RegisterWorkflow(ledgeraccountswf.PreviewAccountDeletion)
	r.RegisterWorkflow(ledgerportfoliowf.RegisterInvestmentTrade)
	r.RegisterWorkflow(ledgerportfoliowf.PositionAt)
	r.RegisterWorkflow(ledgerportfoliowf.RegisterPositionMovement)
	r.RegisterWorkflow(ledgerportfoliowf.ReversePositionMovement)
	r.RegisterWorkflow(ledgerportfoliowf.DeletePositionMovement)
	r.RegisterWorkflow(ledgerportfoliowf.ListPositionMovements)
	r.RegisterWorkflow(ledgerportfoliowf.RegisterVest)
	r.RegisterWorkflow(ledgerreportingwf.FinancialAccountBalances)
	r.RegisterWorkflow(ledgerreportingwf.ListPositionsAt)
	r.RegisterWorkflow(ledgerreportingwf.ListPostings)
	r.RegisterWorkflow(ledgerreportingwf.TrialBalanceAt)
	r.RegisterWorkflow(ledgerreportingwf.CategoryTotals)
	r.RegisterWorkflow(ledgerreportingwf.NetWorthSeries)
}
