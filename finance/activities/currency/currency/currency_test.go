package currency

import (
	"testing"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite
	ca *Currency
}

func (s *UnitTestSuite) SetupTest() {
	s.SetupTestEnvironment()
	s.ca = &Currency{
		Pools: tenancy.Static(s.DB),
	}
	s.Env.RegisterActivity(s.ca)
}

func (s *UnitTestSuite) BeforeTest(_, _ string) {
	sqllib.InitDatabaseT(s.T(), s.DB)
}

func (s *UnitTestSuite) AfterTest(_, _ string) {
	s.DropAllTables()
}

func TestUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}
