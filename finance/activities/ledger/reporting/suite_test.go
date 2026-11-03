package reporting

import (
	"testing"

	ledgerjournal "github.com/logicorg/ivfs/finance/activities/ledger/journal"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/suite"
)

type reportingFixture struct {
	Currency               *model.Currency
	Cash                   *model.LedgerAccount
	Expense                *model.LedgerAccount
	BankInstitution        *model.Institution
	BankFinancialAccount   *model.FinancialAccount
	BrokerInstitution      *model.Institution
	BrokerFinancialAccount *model.FinancialAccount
}

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite
	ra *Activities
	ja *ledgerjournal.Activities
}

func (s *UnitTestSuite) SetupTest() {
	s.SetupTestEnvironment()
	s.ra = &Activities{Pools: tenancy.Static(s.DB)}
	s.ja = &ledgerjournal.Activities{Pools: tenancy.Static(s.DB)}
	s.Env.RegisterActivity(s.ra)
	s.Env.RegisterActivity(s.ja)
}

func (s *UnitTestSuite) BeforeTest(_, _ string) { sqllib.InitDatabaseT(s.T(), s.DB) }
func (s *UnitTestSuite) AfterTest(_, _ string)  { s.DropAllTables() }

func (s *UnitTestSuite) buildReportingFixture(t *testing.T) reportingFixture {
	currency := &model.Currency{Code: "EUR", Name: "Euro", Symbol: "€", Precision: 2}
	temporaltesting.Insert(t, s.DB, currency)

	cash := &model.LedgerAccount{CurrencyID: currency.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	expense := &model.LedgerAccount{CurrencyID: currency.ID, Name: "expense", Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(t, s.DB, cash)
	temporaltesting.Insert(t, s.DB, expense)

	bankInstitution := &model.Institution{Name: "Bank", Kind: "bank"}
	temporaltesting.Insert(t, s.DB, bankInstitution)
	bankFinancialAccount := &model.FinancialAccount{InstitutionID: bankInstitution.ID, Name: "Bank EUR"}
	temporaltesting.Insert(t, s.DB, bankFinancialAccount)

	brokerInstitution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(t, s.DB, brokerInstitution)
	brokerFinancialAccount := &model.FinancialAccount{InstitutionID: brokerInstitution.ID, Name: "Broker"}
	temporaltesting.Insert(t, s.DB, brokerFinancialAccount)

	return reportingFixture{
		Currency:               currency,
		Cash:                   cash,
		Expense:                expense,
		BankInstitution:        bankInstitution,
		BankFinancialAccount:   bankFinancialAccount,
		BrokerInstitution:      brokerInstitution,
		BrokerFinancialAccount: brokerFinancialAccount,
	}
}

func Test_UnitTestSuite(t *testing.T) { suite.Run(t, new(UnitTestSuite)) }
