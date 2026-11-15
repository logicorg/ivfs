package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

// renameFixtures are the ledger accounts each rename case starts from.
type renameFixtures struct {
	cash, savings, wallet, groceries, dining *model.LedgerAccount
}

// Test_RenameLedgerAccount_Taken checks that a ledger account cannot take a
// name another account holds within its key, compared case-insensitively
// (ADR 0005), and is left as it was, while a name outside its key, or its own
// name in another casing, is allowed. This covers the category rename, which
// renames the category's ledger account.
func (s *UnitTestSuite) Test_RenameLedgerAccount_Taken() {
	f := s.seedLedgerAccountFixtures()

	tests := map[string]struct {
		target  func(r renameFixtures) *model.LedgerAccount
		newName string
		errStr  string
		stored  string
	}{
		"the name of an account under the same financial account, in the same currency": {
			target:  func(r renameFixtures) *model.LedgerAccount { return r.savings },
			newName: " cash ",
			errStr:  "resource LedgerAccount conflict",
			stored:  "Savings",
		},
		"the name of a top-level category of the same kind and currency": {
			target:  func(r renameFixtures) *model.LedgerAccount { return r.dining },
			newName: "GROCERIES",
			errStr:  "resource LedgerAccount conflict",
			stored:  "Dining",
		},
		"the name of an account in another currency": {
			target:  func(r renameFixtures) *model.LedgerAccount { return r.wallet },
			newName: "Cash",
			stored:  "Cash",
		},
		"its own name in another casing": {
			target:  func(r renameFixtures) *model.LedgerAccount { return r.groceries },
			newName: "groceries",
			stored:  "groceries",
		},
	}

	for name, tc := range tests {
		r := renameFixtures{
			cash:      &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Cash", Kind: model.LedgerAccountAsset},
			savings:   &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.euro.ID, Name: "Savings", Kind: model.LedgerAccountAsset},
			wallet:    &model.LedgerAccount{FinancialAccountID: &f.personal.ID, CurrencyID: f.dollar.ID, Name: "Wallet", Kind: model.LedgerAccountAsset},
			groceries: &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Groceries", Kind: model.LedgerAccountExpense},
			dining:    &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Dining", Kind: model.LedgerAccountExpense},
		}
		for _, account := range []*model.LedgerAccount{r.cash, r.savings, r.wallet, r.groceries, r.dining} {
			temporaltesting.Insert(s.T(), s.DB, account)
		}
		target := tc.target(r)

		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.RenameLedgerAccount, lt.RenameLedgerAccountInput{LedgerAccountID: target.ID, Name: tc.newName})

			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
			} else {
				require.NoError(s.T(), err)
			}
			var stored model.LedgerAccount
			temporaltesting.Retrieve(s.T(), s.DB, &stored, target.ID)
			require.Equal(s.T(), tc.stored, stored.Name)
		})

		s.deleteLedgerAccounts(r.cash.ID, r.savings.ID, r.wallet.ID, r.groceries.ID, r.dining.ID)
	}
}
