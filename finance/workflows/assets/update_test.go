package assets

import (
	"testing"

	"github.com/logicorg/ivfs/finance/activities/asset"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func (s *UpdateWorkflowUnitTestSuite) TestUpdate() {
	tt := []struct {
		name     string
		params   at.UpdateAssetParams
		setupEnv func(env *testsuite.TestWorkflowEnvironment)
		result   *at.UpdateAssetResult
		wantErr  bool
	}{
		{
			name:   "should fail with invalid findby parameters",
			params: at.UpdateAssetParams{},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.UpdateAsset, mock.Anything, at.UpdateAssetParams{}).
					Return(nil, errors.ErrInvalidParameter{Description: "at least one of ID, Name or Ticker must be provided to find the asset"})
			},
			result:  nil,
			wantErr: true,
		},
		{
			name: "should succeed when asset exists",
			params: at.UpdateAssetParams{
				FindBy: at.RetrieveAssetParams{
					ID: 1,
				},
				Name: ptr.NewPtr("new name"),
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				params := at.UpdateAssetParams{
					FindBy: at.RetrieveAssetParams{
						ID: 1,
					},
					Name: ptr.NewPtr("new name"),
				}
				env.OnActivity(aa.UpdateAsset, mock.Anything, params).
					Return(&at.UpdateAssetResult{ID: 1}, nil)
			},
			result:  &at.UpdateAssetResult{ID: 1},
			wantErr: false,
		},
		{
			name: "should fail when asset does not exist",
			params: at.UpdateAssetParams{
				FindBy: at.RetrieveAssetParams{
					ID: 99,
				},
				Name: ptr.NewPtr("new name"),
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				params := at.UpdateAssetParams{
					FindBy: at.RetrieveAssetParams{
						ID: 99,
					},
					Name: ptr.NewPtr("new name"),
				}
				env.OnActivity(aa.UpdateAsset, mock.Anything, params).
					Return(nil, errors.ErrNotFound{ResourceType: "Asset"})
			},
			result:  nil,
			wantErr: true,
		},
	}

	for _, tc := range tt {
		s.T().Run(tc.name, func(t *testing.T) {
			env := s.NewTestWorkflowEnvironment()
			tc.setupEnv(env)

			env.ExecuteWorkflow(UpdateAsset, tc.params)

			err := env.GetWorkflowError()

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				var result *at.UpdateAssetResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				assert.Equal(t, tc.result, result)
			}
		})
	}
}
