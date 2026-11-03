package conversion

import (
	"testing"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite
	co *Conversion
}

func (s *UnitTestSuite) SetupTest() {
	s.SetupTestEnvironment()
	s.co = &Conversion{
		Pools: tenancy.Static(s.DB),
	}
	s.Env.RegisterActivity(s.co)
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
