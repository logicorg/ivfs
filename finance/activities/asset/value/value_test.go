package value

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
	av *AssetValueActivity

	eur *model.Currency
}

func (s *UnitTestSuite) SetupTest() {
	s.SetupTestEnvironment()
	s.av = &AssetValueActivity{Pools: tenancy.Static(s.DB)}
	s.Env.RegisterActivity(s.av)
}

func (s *UnitTestSuite) BeforeTest(_, _ string) {
	sqllib.InitDatabaseT(s.T(), s.DB)

	s.eur = testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, s.eur)
}

func (s *UnitTestSuite) AfterTest(_, _ string) {
	s.DropAllTables()
}

func Test_UnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}
