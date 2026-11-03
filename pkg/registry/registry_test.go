package registry_test

import (
	"testing"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite
}

func (s *UnitTestSuite) BeforeTest(_, _ string) {
	require.NoError(s.T(), registry.Init(s.T().Context(), s.DB))
}

func (s *UnitTestSuite) AfterTest(_, _ string) {
	_, err := s.DB.ExecContext(s.T().Context(), "DROP SCHEMA IF EXISTS iam CASCADE")
	require.NoError(s.T(), err)
}

func Test_UnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}
