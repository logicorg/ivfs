package institution

import (
	"testing"

	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type WorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestWorkflowUnitTestSuite(t *testing.T) { suite.Run(t, new(WorkflowUnitTestSuite)) }

func (s *WorkflowUnitTestSuite) TestRetrieveInstitution() {
	institutionModel := &model.Institution{ID: 7, Name: "Revolut", Kind: "bank"}

	env := s.NewTestWorkflowEnvironment()
	aa := institutionactivity.Institution{}

	retrieveParams := it.RetrieveInstitutionParams{ID: institutionModel.ID}
	env.OnActivity(aa.Retrieve, mock.Anything, retrieveParams).Return(&it.RetrieveInstitutionResult{Institution: institutionModel}, nil)

	env.ExecuteWorkflow(RetrieveInstitution, retrieveParams)

	require.NoError(s.T(), env.GetWorkflowError())
	var retrieved *it.RetrieveInstitutionResult
	require.NoError(s.T(), env.GetWorkflowResult(&retrieved))
	assert.Equal(s.T(), institutionModel, retrieved.Institution)
}
