package portfolio

import (
	"testing"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite

	pa *Activities

	// eur is the Euro Currency insterted in the DB.
	eur *model.Currency
	// financialAccount is the FinancialAccount in the DB.
	financialAccount *model.FinancialAccount
	// otherAccount is a second FinancialAccount of the same institution, for
	// what moves units between accounts.
	otherAccount *model.FinancialAccount
	// asset is the Asset in the DB.
	asset  *model.Asset
	cash   *model.LedgerAccount
	offset *model.LedgerAccount
	// holding carries the value of the units held, income is where compensation
	// is booked, and tax is where the tax withheld from it is.
	holding, income, tax *model.LedgerAccount
}

func (s *UnitTestSuite) SetupTest() {
	s.SetupTestEnvironment()
	s.pa = &Activities{Pools: tenancy.Static(s.DB)}
	s.Env.RegisterActivity(s.pa)
}

func (s *UnitTestSuite) BeforeTest(_, _ string) {
	sqllib.InitDatabaseT(s.T(), s.DB)

	s.eur = testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, s.eur)

	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	s.financialAccount = &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	temporaltesting.Insert(s.T(), s.DB, s.financialAccount)
	s.otherAccount = &model.FinancialAccount{InstitutionID: institution.ID, Name: "Second broker account"}
	temporaltesting.Insert(s.T(), s.DB, s.otherAccount)

	s.asset = &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, s.asset)

	s.cash = &model.LedgerAccount{FinancialAccountID: &s.financialAccount.ID, CurrencyID: s.eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	s.offset = &model.LedgerAccount{CurrencyID: s.eur.ID, Name: "investment clearing", Kind: model.LedgerAccountAsset}
	temporaltesting.Insert(s.T(), s.DB, s.cash)
	temporaltesting.Insert(s.T(), s.DB, s.offset)

	s.holding = &model.LedgerAccount{FinancialAccountID: &s.financialAccount.ID, CurrencyID: s.eur.ID, Name: "investments", Kind: model.LedgerAccountAsset}
	s.income = &model.LedgerAccount{CurrencyID: s.eur.ID, Name: "compensation", Kind: model.LedgerAccountIncome}
	s.tax = &model.LedgerAccount{CurrencyID: s.eur.ID, Name: "tax withheld", Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, s.holding)
	temporaltesting.Insert(s.T(), s.DB, s.income)
	temporaltesting.Insert(s.T(), s.DB, s.tax)
}
func (s *UnitTestSuite) AfterTest(_, _ string) { s.DropAllTables() }

func Test_UnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}
