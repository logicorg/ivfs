package accounts

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/deletion"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// movementGroupsForDeletionFixture inserts what the cases of
// HandleMovementGroupsForDeletion build on. Kept groups stay behind, so every
// case inserts accounts of its own names and the tables are dropped after the
// test.
type movementGroupsForDeletionFixture struct {
	s           *UnitTestSuite
	currency    *model.Currency
	institution *model.Institution
	asset       *model.Asset
	// broker is a financial account that survives every case.
	broker *model.FinancialAccount
	keys   int
}

func newMovementGroupsForDeletionFixture(s *UnitTestSuite) *movementGroupsForDeletionFixture {
	t := s.T()
	f := &movementGroupsForDeletionFixture{
		s:           s,
		currency:    testhelpers.Euro(),
		institution: &model.Institution{Name: "Revolut", Kind: "bank"},
		asset:       &model.Asset{Name: "AAPL", Kind: model.AssetKindStock},
	}
	temporaltesting.Insert(t, s.DB, f.currency)
	temporaltesting.Insert(t, s.DB, f.institution)
	temporaltesting.Insert(t, s.DB, f.asset)
	f.broker = f.financialAccount(t, "Movement groups broker")
	return f
}

func (f *movementGroupsForDeletionFixture) financialAccount(t *testing.T, name string) *model.FinancialAccount {
	account := &model.FinancialAccount{InstitutionID: f.institution.ID, Name: name}
	temporaltesting.Insert(t, f.s.DB, account)
	return account
}

func (f *movementGroupsForDeletionFixture) strayFinancialAccount(t *testing.T) *model.FinancialAccount {
	var id int64
	require.NoError(t, f.s.DB.RunInTx(t.Context(), nil, func(ctx context.Context, tx bun.Tx) (err error) {
		id, err = deletion.StrayFinancialAccountTx(ctx, tx)
		return err
	}))
	account := &model.FinancialAccount{}
	temporaltesting.Retrieve(t, f.s.DB, account, id)
	return account
}

// leg is units of the asset in account.
type leg struct {
	account *model.FinancialAccount
	units   int64
}

// group records an entry-less group of kind under a key of its own, a leg per
// line, and returns its movements.
func (f *movementGroupsForDeletionFixture) group(t *testing.T, kind model.PositionMovementKind, legs ...leg) []*model.PositionMovement {
	f.keys++
	key := fmt.Sprintf("deletion-group-%d", f.keys)
	movements := make([]*model.PositionMovement, len(legs))
	for n, leg := range legs {
		movements[n] = &model.PositionMovement{
			Kind: kind, FinancialAccountID: leg.account.ID, AssetID: f.asset.ID, UnitsDelta: decimal.NewFromInt(leg.units),
			OccurredAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), IdempotencyKey: &key, LineNumber: n + 1,
		}
		temporaltesting.InsertPositionMovement(t, f.s.DB, movements[n])
	}
	return movements
}

// reversal records the group that reverses group, leg by leg.
func (f *movementGroupsForDeletionFixture) reversal(t *testing.T, group []*model.PositionMovement) []*model.PositionMovement {
	f.keys++
	key := fmt.Sprintf("deletion-group-%d", f.keys)
	movements := make([]*model.PositionMovement, len(group))
	for n, movement := range group {
		movements[n] = &model.PositionMovement{
			Kind: movement.Kind, FinancialAccountID: movement.FinancialAccountID, AssetID: movement.AssetID, UnitsDelta: movement.UnitsDelta.Neg(),
			OccurredAt: movement.OccurredAt, IdempotencyKey: &key, LineNumber: n + 1, ReversalOfMovementID: &movement.ID,
		}
		temporaltesting.InsertPositionMovement(t, f.s.DB, movements[n])
	}
	return movements
}

// groupOutcome is what a batch should leave of a group: nothing, or legs in
// the named financial accounts, in line order.
type groupOutcome struct {
	group   []*model.PositionMovement
	deleted bool
	legs    []string
}

func (f *movementGroupsForDeletionFixture) requireOutcome(t *testing.T, want groupOutcome) {
	t.Helper()

	key := *want.group[0].IdempotencyKey
	var movements []model.PositionMovement
	require.NoError(t, f.s.DB.NewSelect().Model(&movements).Relation("FinancialAccount").
		Where("pm.idempotency_key = ?", key).OrderExpr("pm.line_number").
		Scan(t.Context()))
	if want.deleted {
		require.Empty(t, movements, "group %s is deleted", key)
		return
	}
	var legs []string
	for _, movement := range movements {
		legs = append(legs, movement.FinancialAccount.Name)
	}
	require.Equal(t, want.legs, legs, "the financial accounts of group %s", key)
}

func (s *UnitTestSuite) TestHandleMovementGroupsForDeletion() {
	f := newMovementGroupsForDeletionFixture(s)
	missing := int64(999999)
	financialTarget := func(account *model.FinancialAccount) lt.DeletionTarget {
		return lt.DeletionTarget{FinancialAccountID: &account.ID}
	}

	tests := map[string]struct {
		// setup inserts the case's accounts and groups, and returns what to empty
		// and what should be left of the groups.
		setup func(t *testing.T) (lt.DeletionTarget, []groupOutcome)
		// limit is 100 unless set.
		limit   int
		handled int
		errStr  string
	}{
		"a transfer to a surviving account is kept, with its leg on the stray": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Schwab")
				transfer := f.group(t, model.PositionMovementTransfer, leg{account, -5}, leg{f.broker, 5})
				return financialTarget(account), []groupOutcome{{group: transfer, legs: []string{lt.UntrackedAccountsName, "Movement groups broker"}}}
			},
			handled: 1,
		},
		"a transfer within the account is deleted": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Degiro")
				transfer := f.group(t, model.PositionMovementTransfer, leg{account, -5}, leg{account, 5})
				return financialTarget(account), []groupOutcome{{group: transfer, deleted: true}}
			},
			handled: 1,
		},
		"a transfer to the stray is deleted": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Trade Republic")
				transfer := f.group(t, model.PositionMovementTransfer, leg{account, -5}, leg{f.strayFinancialAccount(t), 5})
				return financialTarget(account), []groupOutcome{{group: transfer, deleted: true}}
			},
			handled: 1,
		},
		"an adjustment in the account is deleted": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Fineco")
				adjustment := f.group(t, model.PositionMovementAdjustment, leg{account, 5})
				return financialTarget(account), []groupOutcome{{group: adjustment, deleted: true}}
			},
			handled: 1,
		},
		"a split across a surviving account is kept, without its leg in the account": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "IBKR")
				split := f.group(t, model.PositionMovementSplit, leg{account, 5}, leg{f.broker, 10})
				return financialTarget(account), []groupOutcome{{group: split, legs: []string{"Movement groups broker"}}}
			},
			handled: 1,
		},
		"an adjustment across a surviving account is kept, without its leg in the account": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Etoro")
				adjustment := f.group(t, model.PositionMovementAdjustment, leg{f.broker, 2}, leg{account, 3})
				return financialTarget(account), []groupOutcome{{group: adjustment, legs: []string{"Movement groups broker"}}}
			},
			handled: 1,
		},
		"a kept transfer and its reversal both move to the stray": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Scalable")
				transfer := f.group(t, model.PositionMovementTransfer, leg{account, -5}, leg{f.broker, 5})
				legs := []string{lt.UntrackedAccountsName, "Movement groups broker"}
				return financialTarget(account), []groupOutcome{{group: transfer, legs: legs}, {group: f.reversal(t, transfer), legs: legs}}
			},
			handled: 2,
		},
		"a kept split and its reversal both lose their legs in the account": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Saxo")
				split := f.group(t, model.PositionMovementSplit, leg{account, 5}, leg{f.broker, 10})
				legs := []string{"Movement groups broker"}
				return financialTarget(account), []groupOutcome{{group: split, legs: legs}, {group: f.reversal(t, split), legs: legs}}
			},
			handled: 2,
		},
		"an adjustment and its reversal are deleted together": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Directa")
				adjustment := f.group(t, model.PositionMovementAdjustment, leg{account, 5})
				return financialTarget(account), []groupOutcome{{group: adjustment, deleted: true}, {group: f.reversal(t, adjustment), deleted: true}}
			},
			handled: 2,
		},
		"a batch handles up to its limit, in the order the groups were written": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Bux")
				first := f.group(t, model.PositionMovementAdjustment, leg{account, 5})
				second := f.group(t, model.PositionMovementAdjustment, leg{account, 3})
				return financialTarget(account), []groupOutcome{{group: first, deleted: true}, {group: second, legs: []string{"Bux"}}}
			},
			limit:   1,
			handled: 1,
		},
		"a batch at its limit deletes a group with its reversal": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Freedom24")
				adjustment := f.group(t, model.PositionMovementAdjustment, leg{account, 5})
				return financialTarget(account), []groupOutcome{{group: adjustment, deleted: true}, {group: f.reversal(t, adjustment), deleted: true}}
			},
			limit:   1,
			handled: 1,
		},
		"a group in another account is left alone": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Moneyfarm")
				adjustment := f.group(t, model.PositionMovementAdjustment, leg{f.broker, 5})
				return financialTarget(account), []groupOutcome{{group: adjustment, legs: []string{"Movement groups broker"}}}
			},
		},
		"a trade is left to the entries": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				account := f.financialAccount(t, "Revolut trading")
				investments := &model.LedgerAccount{Name: "Revolut trading investments", CurrencyID: f.currency.ID, Kind: model.LedgerAccountAsset, FinancialAccountID: &account.ID}
				salary := &model.LedgerAccount{Name: "Revolut trading salary", CurrencyID: f.currency.ID, Kind: model.LedgerAccountIncome}
				temporaltesting.Insert(t, s.DB, investments)
				temporaltesting.Insert(t, s.DB, salary)
				temporaltesting.InsertPostedJournalEntryWithMovements(t, s.DB, &model.JournalEntry{OccurredAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)}, []model.Posting{
					{LedgerAccountID: investments.ID, Amount: decimal.NewFromInt(100), Side: model.PostingDebit},
					{LedgerAccountID: salary.ID, Amount: decimal.NewFromInt(100), Side: model.PostingCredit},
				}, []*model.PositionMovement{{FinancialAccountID: account.ID, AssetID: f.asset.ID, UnitsDelta: decimal.NewFromInt(5)}})
				return financialTarget(account), nil
			},
		},
		"a ledger account has no groups": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				cash := &model.LedgerAccount{Name: "Movement groups cash", CurrencyID: f.currency.ID, Kind: model.LedgerAccountAsset}
				temporaltesting.Insert(t, s.DB, cash)
				return lt.DeletionTarget{LedgerAccountID: &cash.ID}, nil
			},
		},
		"the stray financial account with groups": {
			setup: func(t *testing.T) (lt.DeletionTarget, []groupOutcome) {
				stray := f.strayFinancialAccount(t)
				adjustment := f.group(t, model.PositionMovementAdjustment, leg{stray, 5})
				return financialTarget(stray), []groupOutcome{{group: adjustment, legs: []string{lt.UntrackedAccountsName}}}
			},
			errStr: `"Untracked accounts" holds the lines of deleted accounts and cannot be deleted while it has postings or movements`,
		},
		"a financial account that does not exist": {
			setup: func(*testing.T) (lt.DeletionTarget, []groupOutcome) {
				return lt.DeletionTarget{FinancialAccountID: &missing}, nil
			},
		},
		"a limit that is not positive": {
			setup: func(*testing.T) (lt.DeletionTarget, []groupOutcome) {
				return lt.DeletionTarget{FinancialAccountID: &missing}, nil
			},
			limit:  -1,
			errStr: "a deletion batch limit must be positive",
		},
		"both a ledger and a financial account": {
			setup: func(*testing.T) (lt.DeletionTarget, []groupOutcome) {
				return lt.DeletionTarget{LedgerAccountID: &missing, FinancialAccountID: &missing}, nil
			},
			errStr: "a deletion target is either a ledger account or a financial account",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			t := s.T()
			target, outcomes := tc.setup(t)
			limit := tc.limit
			if limit == 0 {
				limit = 100
			}

			result, err := s.Env.ExecuteActivity(s.aa.HandleMovementGroupsForDeletion, lt.DeletionBatchInput{Target: target, Limit: limit})

			if tc.errStr != "" {
				require.ErrorContains(t, err, tc.errStr)
			} else {
				require.NoError(t, err)
				var got lt.DeletionBatchResult
				require.NoError(t, result.Get(&got))
				require.Equal(t, tc.handled, got.Handled)
			}
			for _, outcome := range outcomes {
				f.requireOutcome(t, outcome)
			}
		})
	}
}

// A batch run again after it handled everything handles nothing, and leaves
// the kept groups' legs where the first run left them.
func (s *UnitTestSuite) TestHandleMovementGroupsForDeletion_Idempotency() {
	t := s.T()
	f := newMovementGroupsForDeletionFixture(s)
	account := f.financialAccount(t, "Revolut stocks")
	outcomes := []groupOutcome{
		{group: f.group(t, model.PositionMovementAdjustment, leg{account, 5}), deleted: true},
		{group: f.group(t, model.PositionMovementTransfer, leg{account, -5}, leg{f.broker, 5}), legs: []string{lt.UntrackedAccountsName, "Movement groups broker"}},
		{group: f.group(t, model.PositionMovementSplit, leg{account, 5}, leg{f.broker, 10}), legs: []string{"Movement groups broker"}},
	}
	input := lt.DeletionBatchInput{Target: lt.DeletionTarget{FinancialAccountID: &account.ID}, Limit: 100}

	for run, handled := range []int{3, 0} {
		result, err := s.Env.ExecuteActivity(s.aa.HandleMovementGroupsForDeletion, input)
		require.NoError(t, err)
		var got lt.DeletionBatchResult
		require.NoError(t, result.Get(&got))
		require.Equal(t, handled, got.Handled, "run %d", run+1)
		for _, outcome := range outcomes {
			f.requireOutcome(t, outcome)
		}
	}
}
