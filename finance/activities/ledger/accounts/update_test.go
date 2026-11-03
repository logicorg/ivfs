package accounts

import (
	"fmt"
	"time"

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

// moveFixtures are the categories each move case starts from: Food and
// Household at the top level, each with a Supermarket, Household with a Bakery
// too, and another Supermarket at the top level.
type moveFixtures struct {
	foodSupermarket, householdSupermarket, bakery, topSupermarket *model.LedgerAccount
}

// Test_MoveLedgerAccount_Taken checks that a ledger account cannot move next to
// an account with the same name, kind, and currency (ADR 0005), and is left
// where it was, while moving next to accounts with other names is allowed.
// This covers the category move, which moves the category's ledger account.
func (s *UnitTestSuite) Test_MoveLedgerAccount_Taken() {
	f := s.seedLedgerAccountFixtures()

	tests := map[string]struct {
		target func(m moveFixtures) *model.LedgerAccount
		// parent is where the target moves to, nil for the top level.
		parent *int64
		errStr string
		// stayed is the parent the target has afterwards.
		stayed func(m moveFixtures) *int64
	}{
		"next to a child with the same name": {
			target: func(m moveFixtures) *model.LedgerAccount { return m.householdSupermarket },
			parent: &f.food.ID,
			errStr: "resource LedgerAccount conflict",
			stayed: func(moveFixtures) *int64 { return &f.household.ID },
		},
		"to the top level, next to a top-level account with the same name": {
			target: func(m moveFixtures) *model.LedgerAccount { return m.foodSupermarket },
			parent: nil,
			errStr: "resource LedgerAccount conflict",
			stayed: func(moveFixtures) *int64 { return &f.food.ID },
		},
		"next to children with other names": {
			target: func(m moveFixtures) *model.LedgerAccount { return m.bakery },
			parent: &f.food.ID,
			stayed: func(moveFixtures) *int64 { return &f.food.ID },
		},
	}

	for name, tc := range tests {
		m := moveFixtures{
			foodSupermarket:      &model.LedgerAccount{ParentID: &f.food.ID, CurrencyID: f.euro.ID, Name: "Supermarket", Kind: model.LedgerAccountExpense},
			householdSupermarket: &model.LedgerAccount{ParentID: &f.household.ID, CurrencyID: f.euro.ID, Name: "Supermarket", Kind: model.LedgerAccountExpense},
			bakery:               &model.LedgerAccount{ParentID: &f.household.ID, CurrencyID: f.euro.ID, Name: "Bakery", Kind: model.LedgerAccountExpense},
			topSupermarket:       &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Supermarket", Kind: model.LedgerAccountExpense},
		}
		for _, account := range []*model.LedgerAccount{m.foodSupermarket, m.householdSupermarket, m.bakery, m.topSupermarket} {
			temporaltesting.Insert(s.T(), s.DB, account)
		}
		target := tc.target(m)

		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.MoveLedgerAccount, lt.MoveLedgerAccountInput{LedgerAccountID: target.ID, ParentID: tc.parent})

			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
			} else {
				require.NoError(s.T(), err)
			}
			var stored model.LedgerAccount
			temporaltesting.Retrieve(s.T(), s.DB, &stored, target.ID)
			require.Equal(s.T(), tc.stayed(m), stored.ParentID)
		})

		s.deleteLedgerAccounts(m.foodSupermarket.ID, m.householdSupermarket.ID, m.bakery.ID, m.topSupermarket.ID)
	}
}

// Test_MoveLedgerAccount_ClosedParent checks that a ledger account cannot move
// under a closed parent, failing with ErrResourceClosed, and stays where it was.
func (s *UnitTestSuite) Test_MoveLedgerAccount_ClosedParent() {
	f := s.seedLedgerAccountFixtures()
	closedAt := time.Now().UTC().Truncate(time.Second)
	closed := &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Pets", Kind: model.LedgerAccountExpense, ClosedAt: &closedAt}
	temporaltesting.Insert(s.T(), s.DB, closed)
	bakery := &model.LedgerAccount{ParentID: &f.food.ID, CurrencyID: f.euro.ID, Name: "Bakery", Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, bakery)

	_, err := s.Env.ExecuteActivity(s.aa.MoveLedgerAccount, lt.MoveLedgerAccountInput{LedgerAccountID: bakery.ID, ParentID: &closed.ID})
	require.ErrorContains(s.T(), err, fmt.Sprintf("parent ledger account %d is closed", closed.ID))

	var stored model.LedgerAccount
	temporaltesting.Retrieve(s.T(), s.DB, &stored, bakery.ID)
	require.Equal(s.T(), &f.food.ID, stored.ParentID)
}
