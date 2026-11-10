package journal

import (
	"testing"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/uptrace/bun"
)

func testLedgerAccount(t *testing.T, db *bun.DB, currencyID int64, name string, kind model.LedgerAccountKind) *model.LedgerAccount {
	t.Helper()
	account := &model.LedgerAccount{CurrencyID: currencyID, Name: name, Kind: kind}
	temporaltesting.Insert(t, db, account)
	return account
}

// held is a financial account holding an asset, with the ledger accounts that
// trades of it are booked against.
type held struct {
	account *model.FinancialAccount
	asset   *model.Asset
	debit   *model.LedgerAccount
	credit  *model.LedgerAccount
}

func (s *UnitTestSuite) newHeld() held {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	temporaltesting.Insert(s.T(), s.DB, account)
	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)

	return held{
		account: account,
		asset:   asset,
		debit:   testLedgerAccount(s.T(), s.DB, euro.ID, "cash", model.LedgerAccountAsset),
		credit:  testLedgerAccount(s.T(), s.DB, euro.ID, "clearing", model.LedgerAccountAsset),
	}
}
