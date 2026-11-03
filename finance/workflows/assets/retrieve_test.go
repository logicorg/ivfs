package assets

import (
	"testing"

	"github.com/logicorg/ivfs/finance/activities/asset"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tiendc/go-deepcopy"
	"go.temporal.io/sdk/testsuite"
)

type RetrieveWorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestRetrieveWorkflowUnitTestSuite(t *testing.T) {
	suite.Run(t, new(RetrieveWorkflowUnitTestSuite))
}

func (s *RetrieveWorkflowUnitTestSuite) TestRetrieveAsset() {
	sampleAsset := &model.Asset{ID: 1, Name: "Test Asset"}

	// PopulateExternal must be called on result
	expectedAsset := new(model.Asset)
	require.NoError(s.T(), deepcopy.Copy(expectedAsset, sampleAsset))

	tt := []struct {
		name     string
		params   *at.RetrieveAssetParams
		setupEnv func(env *testsuite.TestWorkflowEnvironment)
		result   *at.RetrieveAssetResult
		wantErr  bool
	}{
		{
			name:   "should fail with invalid parameters",
			params: &at.RetrieveAssetParams{},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.RetrieveAsset, mock.Anything, at.RetrieveAssetParams{}).
					Return(nil, errors.ErrInvalidParameter{Description: "at least one of ID, Name or Ticker must be provided"})
			},
			result:  nil,
			wantErr: true,
		},
		{
			name: "should succeed when asset exists",
			params: &at.RetrieveAssetParams{
				ID: 1,
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.RetrieveAsset, mock.Anything, at.RetrieveAssetParams{ID: 1}).
					Return(&at.RetrieveAssetResult{Asset: sampleAsset}, nil)
			},
			result:  &at.RetrieveAssetResult{Asset: expectedAsset},
			wantErr: false,
		},
		{
			name: "should fail when asset does not exist",
			params: &at.RetrieveAssetParams{
				ID: 99,
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.RetrieveAsset, mock.Anything, at.RetrieveAssetParams{ID: 99}).
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

			env.ExecuteWorkflow(RetrieveAsset, tc.params)

			err := env.GetWorkflowError()

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				var result *at.RetrieveAssetResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				assert.Equal(t, tc.result, result)
			}
		})
	}
}

func (s *RetrieveWorkflowUnitTestSuite) TestRetrieveAllAssets() {
	sampleAssets := []*model.Asset{
		{ID: 1, Name: "Asset 1"},
		{ID: 2, Name: "Asset 2"},
	}

	expectedAssets := make([]*model.Asset, len(sampleAssets))
	require.NoError(s.T(), deepcopy.Copy(&expectedAssets, &sampleAssets))

	tt := []struct {
		name     string
		setupEnv func(env *testsuite.TestWorkflowEnvironment)
		result   *at.RetrieveAllAssetsResult
		wantErr  bool
	}{
		{
			name: "should succeed when assets exist",
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.RetrieveAllAssets, mock.Anything).
					Return(sampleAssets, nil)
			},
			result:  &at.RetrieveAllAssetsResult{Assets: expectedAssets},
			wantErr: false,
		},
		{
			name: "should succeed when no assets exist",
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.RetrieveAllAssets, mock.Anything).
					Return([]*model.Asset{}, nil)
			},
			result:  &at.RetrieveAllAssetsResult{Assets: []*model.Asset{}},
			wantErr: false,
		},
		{
			name: "should fail on activity error",
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				aa := asset.AssetActivity{}
				env.OnActivity(aa.RetrieveAllAssets, mock.Anything).
					Return(nil, assert.AnError)
			},
			result:  nil,
			wantErr: true,
		},
	}

	for _, tc := range tt {
		s.T().Run(tc.name, func(t *testing.T) {
			env := s.NewTestWorkflowEnvironment()
			tc.setupEnv(env)

			env.ExecuteWorkflow(RetrieveAllAssets)

			err := env.GetWorkflowError()

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				var result *at.RetrieveAllAssetsResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				assert.Equal(t, tc.result, result)
			}
		})
	}
}
