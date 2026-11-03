package tags

import (
	"testing"

	"github.com/logicorg/ivfs/finance/activities/tag"
	"github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type UpdateWorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestUpdateWorkflowUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateWorkflowUnitTestSuite))
}

func (s *UpdateWorkflowUnitTestSuite) Test_UpdateTag() {
	env := s.NewTestWorkflowEnvironment()
	params := &tagtypes.UpdateTagParams{
		ID:   1,
		Name: ptr.NewPtr("new-name"),
	}

	ta := tag.Tag{}

	env.OnActivity(ta.UpdateTag, mock.Anything, *params).Return(nil)

	env.ExecuteWorkflow(UpdateTag, params)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
}
