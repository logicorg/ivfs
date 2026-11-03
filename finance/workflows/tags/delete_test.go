package tags

import (
	"testing"

	"github.com/logicorg/ivfs/finance/activities/tag"
	"github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type DeleteWorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestDeleteWorkflowUnitTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteWorkflowUnitTestSuite))
}

func (s *DeleteWorkflowUnitTestSuite) Test_DeleteTag() {
	env := s.NewTestWorkflowEnvironment()
	params := &tagtypes.DeleteTagParams{
		ID: 1,
	}

	activityParams := tagtypes.DeleteTagParams{
		ID: 1,
	}

	ta := tag.Tag{}

	env.OnActivity(ta.DeleteTag, mock.Anything, activityParams).Return(nil)

	env.ExecuteWorkflow(DeleteTag, params)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
}
