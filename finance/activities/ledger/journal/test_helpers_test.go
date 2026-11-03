package journal

import (
	"testing"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

func testLedgerAccount(t *testing.T, db *bun.DB, currencyID int64, name string, kind model.LedgerAccountKind) *model.LedgerAccount {
	t.Helper()
	account := &model.LedgerAccount{CurrencyID: currencyID, Name: name, Kind: kind}
	temporaltesting.Insert(t, db, account)
	return account
}
