package accounts

import (
	"testing"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite
	aa *Activities
}

func (s *UnitTestSuite) SetupTest() {
	s.SetupTestEnvironment()
	s.aa = &Activities{Pools: tenancy.Static(s.DB)}
	s.Env.RegisterActivity(s.aa)
}
func (s *UnitTestSuite) BeforeTest(_, _ string) { sqllib.InitDatabaseT(s.T(), s.DB) }
func (s *UnitTestSuite) AfterTest(_, _ string)  { s.DropAllTables() }
func Test_UnitTestSuite(t *testing.T)           { suite.Run(t, new(UnitTestSuite)) }
