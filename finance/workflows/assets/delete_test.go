package assets

import (
	"testing"

	"github.com/logicorg/ivfs/finance/activities/asset"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func (s *DeleteWorkflowUnitTestSuite) TestDelete() {
	tt := []struct {
		name     string
		params   at.DeleteAssetParams
		setupEnv func(env *testsuite.TestWorkflowEnvironment)
		result   *at.DeleteAssetResult
		wantErr  bool
	}{
		{
			name:   "should fail with invalid parameters",
			params: at.DeleteAssetParams{},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.DeleteAsset, mock.Anything, at.DeleteAssetParams{}).
					Return(nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ID", "Name", "Ticker"}})
			},
			result:  nil,
			wantErr: true,
		},
		{
			name: "should succeed when asset exists",
			params: at.DeleteAssetParams{
				ID: 1,
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.DeleteAsset, mock.Anything, at.DeleteAssetParams{ID: 1}).
					Return(&at.DeleteAssetResult{}, nil)
			},
			result:  &at.DeleteAssetResult{},
			wantErr: false,
		},
	}

	for _, tc := range tt {
		s.T().Run(tc.name, func(t *testing.T) {
			env := s.NewTestWorkflowEnvironment()
			tc.setupEnv(env)

			env.ExecuteWorkflow(DeleteAsset, tc.params)

			err := env.GetWorkflowError()

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				var result *at.DeleteAssetResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				assert.Equal(t, tc.result, result)
			}
		})
	}
}
