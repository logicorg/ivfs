package accounts

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

// Test_ListFinancialAccounts covers filtering (institution id, account name
// substring, institution name substring, and open/closed) and checks that a
// result carries enough of the institution and ledger account data for a
// caller to disambiguate between candidates without a further lookup.
func (s *UnitTestSuite) Test_ListFinancialAccounts() {
	institutionA := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institutionA)
	institutionB := &model.Institution{Name: "IBKR", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institutionB)
	eur := &model.Currency{Code: "EUR", Name: "Euro", Symbol: "€", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, eur)

	open := &model.FinancialAccount{InstitutionID: institutionA.ID, Name: "Revolut EUR"}
	temporaltesting.Insert(s.T(), s.DB, open)
	cash := &model.LedgerAccount{FinancialAccountID: &open.ID, CurrencyID: eur.ID, Name: "Revolut EUR Cash", Kind: model.LedgerAccountAsset}
	temporaltesting.Insert(s.T(), s.DB, cash)
	closedTime := time.Now().UTC()
	closed := &model.FinancialAccount{InstitutionID: institutionA.ID, Name: "Revolut USD", ClosedAt: &closedTime}
	temporaltesting.Insert(s.T(), s.DB, closed)
	other := &model.FinancialAccount{InstitutionID: institutionB.ID, Name: "IBKR Cash"}
	temporaltesting.Insert(s.T(), s.DB, other)

	tests := map[string]struct {
		params lt.ListFinancialAccountsParams
		expIDs []int64
		expErr bool
	}{
		"all open, no filter": {
			params: lt.ListFinancialAccountsParams{},
			expIDs: []int64{open.ID, other.ID},
		},
		"include closed": {
			params: lt.ListFinancialAccountsParams{IncludeClosed: true},
			expIDs: []int64{open.ID, closed.ID, other.ID},
		},
		"filter by institution id": {
			params: lt.ListFinancialAccountsParams{InstitutionID: &institutionA.ID},
			expIDs: []int64{open.ID},
		},
		"filter by name substring, case-insensitive": {
			params: lt.ListFinancialAccountsParams{NameContains: "revolut"},
			expIDs: []int64{open.ID},
		},
		"name substring with no match returns empty": {
			params: lt.ListFinancialAccountsParams{NameContains: "nonexistent"},
			expIDs: []int64{},
		},
		"filter by institution name substring": {
			params: lt.ListFinancialAccountsParams{InstitutionNameContains: "revo"},
			expIDs: []int64{open.ID},
		},
		"institution name substring includes closed when asked": {
			params: lt.ListFinancialAccountsParams{InstitutionNameContains: "revo", IncludeClosed: true},
			expIDs: []int64{open.ID, closed.ID},
		},
		"name and institution filters are ANDed": {
			params: lt.ListFinancialAccountsParams{NameContains: "cash", InstitutionNameContains: "ibkr"},
			expIDs: []int64{other.ID},
		},
		"invalid institution id": {
			params: lt.ListFinancialAccountsParams{InstitutionID: ptr.Int64(0)},
			expErr: true,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.ListFinancialAccounts, tc.params)
			if tc.expErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)

			var result lt.ListFinancialAccountsResult
			require.NoError(s.T(), activityResult.Get(&result))
			gotIDs := make([]int64, 0, len(result.FinancialAccounts))
			for _, a := range result.FinancialAccounts {
				gotIDs = append(gotIDs, a.ID)
			}
			require.ElementsMatch(s.T(), tc.expIDs, gotIDs)
		})
	}
}

// Test_ListFinancialAccounts_LoadsDisambiguatingRelations checks that a
// result carries its institution and its ledger accounts' currency, which a
// caller needs to tell candidates apart without a further round trip.
func (s *UnitTestSuite) Test_ListFinancialAccounts_LoadsDisambiguatingRelations() {
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	eur := &model.Currency{Code: "EUR", Name: "Euro", Symbol: "€", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, eur)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut EUR"}
	temporaltesting.Insert(s.T(), s.DB, account)
	cash := &model.LedgerAccount{FinancialAccountID: &account.ID, CurrencyID: eur.ID, Name: "Revolut EUR Cash", Kind: model.LedgerAccountAsset}
	temporaltesting.Insert(s.T(), s.DB, cash)

	activityResult, err := s.Env.ExecuteActivity(s.aa.ListFinancialAccounts, lt.ListFinancialAccountsParams{})
	require.NoError(s.T(), err)

	var result lt.ListFinancialAccountsResult
	require.NoError(s.T(), activityResult.Get(&result))
	require.Len(s.T(), result.FinancialAccounts, 1)

	got := result.FinancialAccounts[0]
	require.NotNil(s.T(), got.Institution)
	require.Equal(s.T(), "Revolut", got.Institution.Name)
	require.Equal(s.T(), "bank", got.Institution.Kind)
	require.Len(s.T(), got.LedgerAccounts, 1)
	require.Equal(s.T(), model.LedgerAccountAsset, got.LedgerAccounts[0].Kind)
	require.NotNil(s.T(), got.LedgerAccounts[0].Currency)
	require.Equal(s.T(), "EUR", got.LedgerAccounts[0].Currency.Code)
}
