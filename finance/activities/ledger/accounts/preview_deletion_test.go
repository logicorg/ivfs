package accounts

import (
	"fmt"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

// previewOutcome is a preview with its strays as text, "name kind currency
// amount" and "asset units", so amounts compare by value.
type previewOutcome struct {
	deletedEntries, keptEntries               int
	deletedMovementGroups, keptMovementGroups int
	amounts, units, conflicts                 []string
}

func outcomeOf(preview *lt.AccountDeletionPreview) previewOutcome {
	got := previewOutcome{
		deletedEntries: preview.DeletedEntries, keptEntries: preview.KeptEntries,
		deletedMovementGroups: preview.DeletedMovementGroups, keptMovementGroups: preview.KeptMovementGroups,
		conflicts: preview.Conflicts,
	}
	for _, amount := range preview.StrayAmounts {
		got.amounts = append(got.amounts, fmt.Sprintf("%s %s %s %s", amount.Name, amount.Kind, amount.CurrencyCode, amount.Amount))
	}
	for _, units := range preview.StrayUnits {
		got.units = append(got.units, fmt.Sprintf("%s %s", units.AssetName, units.Units))
	}
	return got
}

func (s *UnitTestSuite) TestPreviewAccountDeletion() {
	f := newEntriesForDeletionFixture(s)
	g := &movementGroupsForDeletionFixture{s: s, currency: f.currency, institution: f.institution, asset: f.asset, broker: f.broker}
	missing := int64(999999)
	ledgerInput := func(account *model.LedgerAccount) lt.PreviewAccountDeletionInput {
		return lt.PreviewAccountDeletionInput{LedgerAccountID: &account.ID}
	}
	financialInput := func(account *model.FinancialAccount) lt.PreviewAccountDeletionInput {
		return lt.PreviewAccountDeletionInput{FinancialAccountID: &account.ID}
	}
	// institution inserts an institution of its own with a financial account
	// per name, each with a cash ledger account.
	institution := func(t *testing.T, name string, accounts ...string) (*model.Institution, []*model.LedgerAccount) {
		institution := &model.Institution{Name: name, Kind: "bank"}
		temporaltesting.Insert(t, s.DB, institution)
		var cash []*model.LedgerAccount
		for _, account := range accounts {
			financialAccount := &model.FinancialAccount{InstitutionID: institution.ID, Name: account}
			temporaltesting.Insert(t, s.DB, financialAccount)
			cash = append(cash, f.ledgerAccount(t, account+" cash", model.LedgerAccountAsset, &financialAccount.ID))
		}
		return institution, cash
	}

	tests := map[string]struct {
		// setup inserts the case's accounts and lines, and returns what to preview
		// and what the preview should say.
		setup  func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome)
		errStr string
	}{
		"a ledger account's expense is deleted and its transfer kept": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				wallet := f.ledgerAccount(t, "Wallet", model.LedgerAccountAsset, nil)
				f.entry(t, f.groceries, wallet)
				f.entry(t, wallet, f.bank)
				f.draft(t, wallet, f.bank)
				return ledgerInput(wallet), previewOutcome{
					deletedEntries: 2, keptEntries: 1,
					amounts: []string{"Untracked accounts equity EUR 100"},
				}
			},
		},
		"a category's kept expense lands on uncategorized": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				coffee := f.ledgerAccount(t, "Coffee", model.LedgerAccountExpense, nil)
				f.entry(t, coffee, f.bank)
				return ledgerInput(coffee), previewOutcome{keptEntries: 1, amounts: []string{"Uncategorized expense EUR 100"}}
			},
		},
		"a kept entry and its reversal land nothing": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				loan := f.ledgerAccount(t, "Loan cash", model.LedgerAccountAsset, nil)
				f.reversal(t, f.entry(t, loan, f.bank), loan, f.bank)
				return ledgerInput(loan), previewOutcome{keptEntries: 2}
			},
		},
		"a ledger account without lines": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				return ledgerInput(f.ledgerAccount(t, "Unused cash", model.LedgerAccountAsset, nil)), previewOutcome{}
			},
		},
		"a ledger account with children": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				parent := f.ledgerAccount(t, "Parent cash", model.LedgerAccountAsset, nil)
				child := &model.LedgerAccount{Name: "Parent cash pot", CurrencyID: f.currency.ID, Kind: model.LedgerAccountAsset, ParentID: &parent.ID}
				temporaltesting.Insert(t, s.DB, child)
				return ledgerInput(parent), previewOutcome{
					conflicts: []string{fmt.Sprintf("ledger account %d has child accounts; delete them first", parent.ID)},
				}
			},
		},
		"a shared ledger account with entries": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				f.entry(t, f.bank, f.clearing)
				return ledgerInput(f.clearing), previewOutcome{
					keptEntries: 1,
					amounts:     []string{"Untracked accounts equity EUR -100"},
					conflicts:   []string{`"FX clearing" is shared by the ledger and cannot be deleted while it has postings`},
				}
			},
		},
		"a financial account's trade inside it is deleted and its trade from a surviving balance kept": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				account := f.financialAccount(t, "Degiro")
				investments := f.ledgerAccount(t, "Degiro investments", model.LedgerAccountAsset, &account.ID)
				cash := f.ledgerAccount(t, "Degiro cash", model.LedgerAccountAsset, &account.ID)
				f.entry(t, investments, cash, f.units(account))
				f.entry(t, investments, f.bank, f.units(account))
				return financialInput(account), previewOutcome{
					deletedEntries: 1, keptEntries: 1,
					amounts: []string{"Untracked accounts equity EUR 100"},
					units:   []string{"AAPL 5"},
				}
			},
		},
		"a financial account's groups": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				account := f.financialAccount(t, "IBKR")
				g.group(t, model.PositionMovementTransfer, leg{account, -5}, leg{f.broker, 5})
				g.reversal(t, g.group(t, model.PositionMovementTransfer, leg{account, -3}, leg{f.broker, 3}))
				g.group(t, model.PositionMovementSplit, leg{account, 10})
				g.group(t, model.PositionMovementSplit, leg{account, 2}, leg{f.broker, 2})
				return financialInput(account), previewOutcome{
					deletedMovementGroups: 1, keptMovementGroups: 4,
					units: []string{"AAPL -5"},
				}
			},
		},
		"a financial account with a child outside it": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				account := f.financialAccount(t, "Revolut parent")
				cash := f.ledgerAccount(t, "Revolut parent cash", model.LedgerAccountAsset, &account.ID)
				child := &model.LedgerAccount{Name: "Revolut parent pot", CurrencyID: f.currency.ID, Kind: model.LedgerAccountAsset, ParentID: &cash.ID}
				temporaltesting.Insert(t, s.DB, child)
				return financialInput(account), previewOutcome{
					conflicts: []string{fmt.Sprintf("a ledger account of financial account %d has child accounts outside it; delete or move them first", account.ID)},
				}
			},
		},
		"the stray financial account with entries": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				stray := f.strayFinancialAccount(t)
				f.entry(t, f.groceries, f.salary, f.units(stray))
				return financialInput(stray), previewOutcome{
					deletedEntries: 1,
					conflicts:      []string{`"Untracked accounts" holds the lines of deleted accounts and cannot be deleted while it has postings or movements`},
				}
			},
		},
		"an institution's transfer between its accounts is deleted": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				institution, cash := institution(t, "N26", "N26 main", "N26 savings")
				f.entry(t, cash[1], cash[0])
				f.entry(t, cash[0], f.bank)
				return lt.PreviewAccountDeletionInput{InstitutionID: &institution.ID}, previewOutcome{
					deletedEntries: 1, keptEntries: 1,
					amounts: []string{"Untracked accounts equity EUR 100"},
				}
			},
		},
		"an institution without accounts": {
			setup: func(t *testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				institution, _ := institution(t, "Bunq")
				return lt.PreviewAccountDeletionInput{InstitutionID: &institution.ID}, previewOutcome{}
			},
		},
		"a ledger account that does not exist": {
			setup: func(*testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				return lt.PreviewAccountDeletionInput{LedgerAccountID: &missing}, previewOutcome{}
			},
			errStr: "resource LedgerAccount not found with ID: 999999",
		},
		"a financial account that does not exist": {
			setup: func(*testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				return lt.PreviewAccountDeletionInput{FinancialAccountID: &missing}, previewOutcome{}
			},
			errStr: "resource FinancialAccount not found with ID: 999999",
		},
		"an institution that does not exist": {
			setup: func(*testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				return lt.PreviewAccountDeletionInput{InstitutionID: &missing}, previewOutcome{}
			},
			errStr: "resource Institution not found with ID: 999999",
		},
		"two targets": {
			setup: func(*testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				return lt.PreviewAccountDeletionInput{LedgerAccountID: &missing, InstitutionID: &missing}, previewOutcome{}
			},
			errStr: "a deletion preview takes exactly one of a ledger account, a financial account, or an institution",
		},
		"an id that is not positive": {
			setup: func(*testing.T) (lt.PreviewAccountDeletionInput, previewOutcome) {
				return lt.PreviewAccountDeletionInput{LedgerAccountID: new(int64)}, previewOutcome{}
			},
			errStr: "validation error",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			t := s.T()
			input, want := tc.setup(t)
			entries := f.rows(t)

			result, err := s.Env.ExecuteActivity(s.aa.PreviewAccountDeletion, input)

			if tc.errStr != "" {
				require.ErrorContains(t, err, tc.errStr)
				return
			}
			require.NoError(t, err)
			var preview lt.AccountDeletionPreview
			require.NoError(t, result.Get(&preview))
			require.Equal(t, want, outcomeOf(&preview))
			require.Equal(t, entries, f.rows(t), "a preview changes nothing")
		})
	}
}

// The "Untracked" institution holds the stray financial account, which other
// cases put lines on, so it is previewed in a test of its own.
func (s *UnitTestSuite) TestPreviewAccountDeletion_Untracked() {
	t := s.T()
	f := newEntriesForDeletionFixture(s)
	stray := f.strayFinancialAccount(t)
	f.entry(t, f.groceries, f.salary, f.units(stray))

	result, err := s.Env.ExecuteActivity(s.aa.PreviewAccountDeletion, lt.PreviewAccountDeletionInput{InstitutionID: &stray.InstitutionID})

	require.NoError(t, err)
	var preview lt.AccountDeletionPreview
	require.NoError(t, result.Get(&preview))
	require.Equal(t, previewOutcome{
		deletedEntries: 1,
		conflicts:      []string{`"Untracked" is managed by the ledger and cannot be deleted while it holds financial accounts`},
	}, outcomeOf(&preview))
}

// rows counts the entries, postings and movements.
func (f *entriesForDeletionFixture) rows(t *testing.T) int {
	count := 0
	for _, table := range []any{(*model.JournalEntry)(nil), (*model.Posting)(nil), (*model.PositionMovement)(nil)} {
		n, err := f.s.DB.NewSelect().Model(table).Count(t.Context())
		require.NoError(t, err)
		count += n
	}
	return count
}
