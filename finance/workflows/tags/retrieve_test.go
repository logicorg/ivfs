package tags

import (
	"testing"

	"github.com/logicorg/ivfs/finance/activities/tag"
	"github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

func (s *UnitTestSuite) Test_RetrieveTag() {
	env := s.NewTestWorkflowEnvironment()
	params := &tagtypes.RetrieveTagParams{ID: 1}
	activityResult := &tagtypes.RetrieveTagResult{
		Tag: &model.Tag{ID: 1, Name: "test-tag"},
	}

	ta := tag.Tag{}

	env.OnActivity(ta.RetrieveTag, mock.Anything, tagtypes.RetrieveTagParams{ID: 1}).Return(activityResult, nil)

	env.ExecuteWorkflow(RetrieveTag, params)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result *tagtypes.RetrieveTagResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Require().NotNil(result)
	s.Require().NotNil(result.Tag)
	s.Equal(int64(1), result.Tag.ID)
	s.Equal("test-tag", result.Tag.Name)
}

func (s *UnitTestSuite) Test_ListTags() {
	env := s.NewTestWorkflowEnvironment()
	activityResult := []*model.Tag{
		{ID: 1, Name: "tag1"},
		{ID: 2, Name: "tag2"},
	}

	ta := tag.Tag{}

	env.OnActivity(ta.ListTags, mock.Anything).Return(activityResult, nil)

	in := new(tagtypes.ListTagsParams)
	env.ExecuteWorkflow(ListTags, in)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result *tagtypes.ListTagsResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Require().NotNil(result)
	s.Len(result.Tags, 2)
	s.Equal("tag1", result.Tags[0].Name)
	s.Equal("tag2", result.Tags[1].Name)
}
