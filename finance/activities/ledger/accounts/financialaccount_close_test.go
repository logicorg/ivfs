package accounts

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_CloseFinancialAccount() {
	closedAt := time.Now().UTC().Truncate(time.Second)
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	s.Run("closes an account with no active ledger accounts", func() {
		account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut EUR"}
		temporaltesting.Insert(s.T(), s.DB, account)

		_, err := s.Env.ExecuteActivity(s.aa.CloseFinancialAccount, lt.CloseFinancialAccountInput{
			FinancialAccountID: account.ID,
			ClosedAt:           closedAt,
		})
		require.NoError(s.T(), err)

		var stored model.FinancialAccount
		require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", account.ID).Scan(s.T().Context()))
		require.NotNil(s.T(), stored.ClosedAt)
		require.Equal(s.T(), closedAt, stored.ClosedAt.UTC())
	})

	s.Run("refuses to close with an active linked ledger account", func() {
		currency := &model.Currency{Code: "EUR", Name: "Euro", Symbol: "€", Precision: 2}
		temporaltesting.Insert(s.T(), s.DB, currency)
		account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut EUR 2"}
		temporaltesting.Insert(s.T(), s.DB, account)
		ledgerAccount := &model.LedgerAccount{FinancialAccountID: &account.ID, CurrencyID: currency.ID, Name: "cash", Kind: model.LedgerAccountAsset}
		temporaltesting.Insert(s.T(), s.DB, ledgerAccount)

		_, err := s.Env.ExecuteActivity(s.aa.CloseFinancialAccount, lt.CloseFinancialAccountInput{
			FinancialAccountID: account.ID,
			ClosedAt:           closedAt,
		})
		require.Error(s.T(), err)

		var stored model.FinancialAccount
		require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", account.ID).Scan(s.T().Context()))
		require.Nil(s.T(), stored.ClosedAt)
	})

	s.Run("closes once the linked ledger account is itself closed", func() {
		currency := &model.Currency{Code: "USD", Name: "Dollar", Symbol: "$", Precision: 2}
		temporaltesting.Insert(s.T(), s.DB, currency)
		account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut USD"}
		temporaltesting.Insert(s.T(), s.DB, account)
		laClosedAt := closedAt
		ledgerAccount := &model.LedgerAccount{FinancialAccountID: &account.ID, CurrencyID: currency.ID, Name: "cash", Kind: model.LedgerAccountAsset, ClosedAt: &laClosedAt}
		temporaltesting.Insert(s.T(), s.DB, ledgerAccount)

		_, err := s.Env.ExecuteActivity(s.aa.CloseFinancialAccount, lt.CloseFinancialAccountInput{
			FinancialAccountID: account.ID,
			ClosedAt:           closedAt,
		})
		require.NoError(s.T(), err)
	})

	s.Run("not found", func() {
		_, err := s.Env.ExecuteActivity(s.aa.CloseFinancialAccount, lt.CloseFinancialAccountInput{
			FinancialAccountID: 999999,
			ClosedAt:           closedAt,
		})
		require.Error(s.T(), err)
	})

	s.Run("invalid id", func() {
		_, err := s.Env.ExecuteActivity(s.aa.CloseFinancialAccount, lt.CloseFinancialAccountInput{
			FinancialAccountID: 0,
			ClosedAt:           closedAt,
		})
		require.Error(s.T(), err)
	})
}

func (s *UnitTestSuite) Test_CloseFinancialAccount_Idempotency() {
	closedAt := time.Now().UTC().Truncate(time.Second)
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut EUR"}
	temporaltesting.Insert(s.T(), s.DB, account)

	params := lt.CloseFinancialAccountInput{FinancialAccountID: account.ID, ClosedAt: closedAt}

	_, err := s.Env.ExecuteActivity(s.aa.CloseFinancialAccount, params)
	require.NoError(s.T(), err)
	_, err = s.Env.ExecuteActivity(s.aa.CloseFinancialAccount, params)
	require.NoError(s.T(), err)

	var stored model.FinancialAccount
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", account.ID).Scan(s.T().Context()))
	require.NotNil(s.T(), stored.ClosedAt)
	require.Equal(s.T(), closedAt, stored.ClosedAt.UTC())
}

// A financial account that still holds units of an asset cannot be closed,
// since they would be left under an account that accepts no more writes.
func (s *UnitTestSuite) Test_CloseFinancialAccount_HoldingUnits() {
	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	temporaltesting.Insert(s.T(), s.DB, account)
	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)

	holding := &model.PositionMovement{FinancialAccountID: account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5), OccurredAt: time.Now().UTC().Add(-time.Hour)}
	temporaltesting.InsertPositionMovement(s.T(), s.DB, holding)
	params := lt.CloseFinancialAccountInput{FinancialAccountID: account.ID, ClosedAt: time.Now().UTC()}

	_, err := s.Env.ExecuteActivity(s.aa.CloseFinancialAccount, params)
	require.ErrorContains(s.T(), err, "still holds units")

	var stored model.FinancialAccount
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", account.ID).Scan(s.T().Context()))
	require.Nil(s.T(), stored.ClosedAt)

	temporaltesting.InsertPositionMovement(s.T(), s.DB, &model.PositionMovement{FinancialAccountID: account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(-5), OccurredAt: time.Now().UTC()})

	_, err = s.Env.ExecuteActivity(s.aa.CloseFinancialAccount, params)
	require.NoError(s.T(), err)
}

// Closing has to wait for a trade in progress on the account and then see the
// units it added. Without the lock it would check an empty position, close the
// account, and let the trade commit into a closed account.
func (s *UnitTestSuite) Test_CloseFinancialAccount_WaitsForConcurrentTrade() {
	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	temporaltesting.Insert(s.T(), s.DB, account)
	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)

	trading, err := s.DB.BeginTx(s.T().Context(), nil)
	require.NoError(s.T(), err)
	defer func() { _ = trading.Rollback() }()
	_, err = trading.ExecContext(s.T().Context(), "SELECT id FROM account.financial_account WHERE id = ? FOR NO KEY UPDATE", account.ID)
	require.NoError(s.T(), err)
	_, err = trading.NewInsert().Model(temporaltesting.WithMovementDefaults(&model.PositionMovement{FinancialAccountID: account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5), OccurredAt: time.Now().UTC()})).Exec(s.T().Context())
	require.NoError(s.T(), err)

	closed := make(chan error, 1)
	go func() {
		closed <- s.aa.CloseFinancialAccount(s.T().Context(), lt.CloseFinancialAccountInput{FinancialAccountID: account.ID, ClosedAt: time.Now().UTC()})
	}()

	select {
	case <-closed:
		s.T().Fatal("the account was closed without waiting for the trade in progress")
	case <-time.After(500 * time.Millisecond):
	}

	require.NoError(s.T(), trading.Commit())
	require.ErrorContains(s.T(), <-closed, "still holds units")
}
