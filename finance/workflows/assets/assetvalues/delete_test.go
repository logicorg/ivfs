package assetvalues

import (
	"testing"

	"github.com/logicorg/ivfs/finance/activities/asset/value"
	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
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
		params   avt.DeleteAssetValueParams
		setupEnv func(env *testsuite.TestWorkflowEnvironment)
		result   *avt.DeleteAssetValueResult
		wantErr  bool
	}{
		{
			name:   "should fail with invalid parameters",
			params: avt.DeleteAssetValueParams{},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				ava := value.AssetValueActivity{}
				env.OnActivity(ava.DeleteAssetValue, mock.Anything, avt.DeleteAssetValueParams{}).
					Return(nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ID"}})
			},
			result:  nil,
			wantErr: true,
		},
		{
			name: "should succeed when asset value exists",
			params: avt.DeleteAssetValueParams{
				ID: 1,
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				ava := value.AssetValueActivity{}
				env.OnActivity(ava.DeleteAssetValue, mock.Anything, avt.DeleteAssetValueParams{ID: 1}).
					Return(&avt.DeleteAssetValueResult{}, nil)
			},
			result:  &avt.DeleteAssetValueResult{},
			wantErr: false,
		},
	}

	for _, tc := range tt {
		s.T().Run(tc.name, func(t *testing.T) {
			env := s.NewTestWorkflowEnvironment()
			tc.setupEnv(env)

			env.ExecuteWorkflow(DeleteAssetValue, tc.params)

			err := env.GetWorkflowError()

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				var result *avt.DeleteAssetValueResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				assert.Equal(t, tc.result, result)
			}
		})
	}
}
