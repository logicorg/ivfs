package fx

import (
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite

	fx *Activities

	eur *model.Currency
	usd *model.Currency

	revolutUSD  *model.LedgerAccount
	revolutEUR  *model.LedgerAccount
	usdClearing *model.LedgerAccount
	eurClearing *model.LedgerAccount
	usdFees     *model.LedgerAccount
	eurFees     *model.LedgerAccount
	usdIncome   *model.LedgerAccount
	savingsUSD  *model.LedgerAccount
	savingsEUR  *model.LedgerAccount
	closedUSD   *model.LedgerAccount
}

func (s *UnitTestSuite) SetupTest() {
	s.SetupTestEnvironment()
	s.fx = &Activities{Pools: tenancy.Static(s.DB)}
	s.Env.RegisterActivity(s.fx)
}

func (s *UnitTestSuite) BeforeTest(_, _ string) {
	sqllib.InitDatabaseT(s.T(), s.DB)

	s.eur = testhelpers.Euro()
	s.usd = testhelpers.USDollar()
	temporaltesting.Insert(s.T(), s.DB, s.eur)
	temporaltesting.Insert(s.T(), s.DB, s.usd)

	s.revolutUSD = s.insertAccount(s.usd, "Revolut USD", model.LedgerAccountAsset)
	s.revolutEUR = s.insertAccount(s.eur, "Revolut EUR", model.LedgerAccountAsset)
	s.usdClearing = s.insertAccount(s.usd, "FX-Clearing USD", model.LedgerAccountAsset)
	s.eurClearing = s.insertAccount(s.eur, "FX-Clearing EUR", model.LedgerAccountAsset)
	s.usdFees = s.insertAccount(s.usd, "FX-Fees USD", model.LedgerAccountExpense)
	s.eurFees = s.insertAccount(s.eur, "FX-Fees EUR", model.LedgerAccountExpense)
	s.usdIncome = s.insertAccount(s.usd, "Interest USD", model.LedgerAccountIncome)
	s.savingsUSD = s.insertAccount(s.usd, "Savings USD", model.LedgerAccountAsset)
	s.savingsEUR = s.insertAccount(s.eur, "Savings EUR", model.LedgerAccountAsset)

	s.closedUSD = s.insertAccount(s.usd, "Closed USD", model.LedgerAccountAsset)
	closedAt := time.Now().UTC()
	s.closedUSD.ClosedAt = &closedAt
	_, err := s.DB.NewUpdate().Model(s.closedUSD).Column("closed_at").WherePK().Exec(s.T().Context())
	s.Require().NoError(err)
}

func (s *UnitTestSuite) AfterTest(_, _ string) { s.DropAllTables() }

func Test_UnitTestSuite(t *testing.T) { suite.Run(t, new(UnitTestSuite)) }

func (s *UnitTestSuite) insertAccount(currency *model.Currency, name string, kind model.LedgerAccountKind) *model.LedgerAccount {
	account := &model.LedgerAccount{CurrencyID: currency.ID, Name: name, Kind: kind}
	temporaltesting.Insert(s.T(), s.DB, account)
	return account
}

// adrExample is the conversion from ADR 0001: USD100 leave Revolut USD, of
// which USD1 is a fee, and EUR90 arrive in Revolut EUR.
func (s *UnitTestSuite) adrExample() lt.RegisterFXConversionInput {
	return lt.RegisterFXConversionInput{
		IdempotencyKey:                     "fx-usd-eur-1",
		OccurredAt:                         time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
		SourceLedgerAccountID:              s.revolutUSD.ID,
		SourceClearingLedgerAccountID:      s.usdClearing.ID,
		SourceAmount:                       decimal.NewFromInt(100),
		DestinationLedgerAccountID:         s.revolutEUR.ID,
		DestinationClearingLedgerAccountID: s.eurClearing.ID,
		DestinationAmount:                  decimal.NewFromInt(90),
		Fee:                                &lt.FXFee{LedgerAccountID: s.usdFees.ID, Amount: decimal.NewFromInt(1)},
	}
}
