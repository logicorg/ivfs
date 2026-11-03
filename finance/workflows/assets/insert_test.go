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

type InsertWorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestInsertWorkflowUnitTestSuite(t *testing.T) {
	suite.Run(t, new(InsertWorkflowUnitTestSuite))
}

func (s *InsertWorkflowUnitTestSuite) TestInsert() {
	tt := []struct {
		name     string
		params   at.InsertAssetParams
		setupEnv func(env *testsuite.TestWorkflowEnvironment)
		result   *at.InsertAssetResult
		wantErr  bool
	}{
		{
			name: "should fail with invalid parameters",
			params: at.InsertAssetParams{
				Name: "",
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.InsertAsset, mock.Anything, at.InsertAssetParams{Name: ""}).
					Return(nil, errors.ErrMissingRequiredParameter{Parameters: []string{"Name"}})
			},
			result:  nil,
			wantErr: true,
		},
		{
			name: "should succeed when asset is new",
			params: at.InsertAssetParams{
				Name: "new asset",
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.InsertAsset, mock.Anything, at.InsertAssetParams{Name: "new asset"}).
					Return(&at.InsertAssetResult{ID: 1}, nil)
			},
			result:  &at.InsertAssetResult{ID: 1},
			wantErr: false,
		},
		{
			name: "should succeed when asset already exists",
			params: at.InsertAssetParams{
				Name: "existing asset",
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.InsertAsset, mock.Anything, at.InsertAssetParams{Name: "existing asset"}).
					Return(&at.InsertAssetResult{ID: 2}, nil)
			},
			result:  &at.InsertAssetResult{ID: 2},
			wantErr: false,
		},
	}

	for _, tc := range tt {
		s.T().Run(tc.name, func(t *testing.T) {
			env := s.NewTestWorkflowEnvironment()
			tc.setupEnv(env)

			env.ExecuteWorkflow(InsertAsset, tc.params)

			err := env.GetWorkflowError()

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				var result *at.InsertAssetResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				assert.Equal(t, tc.result, result)
			}
		})
	}
}
