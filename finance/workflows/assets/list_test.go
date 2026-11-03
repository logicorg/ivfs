package assets

import (
	"testing"

	"github.com/logicorg/ivfs/finance/activities/asset"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type ListWorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestListWorkflowUnitTestSuite(t *testing.T) {
	suite.Run(t, new(ListWorkflowUnitTestSuite))
}

// TestListAssets checks that the workflow is a thin wrapper around the
// ListAssets activity: params pass through unchanged and the activity's
// result is returned as-is.
func (s *ListWorkflowUnitTestSuite) TestListAssets() {
	sampleAssets := []*model.Asset{{ID: 1, Name: "Apple Inc."}}

	params := at.ListAssetsParams{NameContains: "apple"}
	env := s.NewTestWorkflowEnvironment()
	aa := asset.AssetActivity{}
	env.OnActivity(aa.ListAssets, mock.Anything, params).
		Return(&at.ListAssetsResult{Assets: sampleAssets}, nil)

	env.ExecuteWorkflow(ListAssets, params)
	require.NoError(s.T(), env.GetWorkflowError())

	var result *at.ListAssetsResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), sampleAssets, result.Assets)
}
