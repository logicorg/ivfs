package assetvalues

import (
	"testing"
	"time"

	"github.com/logicorg/ivfs/finance/activities/asset/value"
	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type RetrieveWorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestRetrieveWorkflowUnitTestSuite(t *testing.T) {
	suite.Run(t, new(RetrieveWorkflowUnitTestSuite))
}

func (s *RetrieveWorkflowUnitTestSuite) TestRetrieve() {
	now := time.Now().UTC()
	tt := []struct {
		name     string
		params   avt.RetrieveAssetValueParams
		setupEnv func(env *testsuite.TestWorkflowEnvironment)
		result   *avt.RetrieveAssetValueResult
		wantErr  bool
	}{
		{
			name:   "should fail with invalid parameters",
			params: avt.RetrieveAssetValueParams{},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				ava := value.AssetValueActivity{}
				env.OnActivity(ava.RetrieveAssetValue, mock.Anything, avt.RetrieveAssetValueParams{}).
					Return(nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ID", "AssetID"}})
			},
			result:  nil,
			wantErr: true,
		},
		{
			name: "should succeed with valid params",
			params: avt.RetrieveAssetValueParams{
				ID: 1,
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				ava := value.AssetValueActivity{}
				env.OnActivity(ava.RetrieveAssetValue, mock.Anything, avt.RetrieveAssetValueParams{
					ID: 1,
				}).
					Return(&avt.RetrieveAssetValueResult{AssetValue: &model.AssetValue{
						ID:         1,
						AssetID:    1,
						Value:      decimal.NewFromInt(123),
						ObservedAt: now,
					}}, nil)
			},
			result: &avt.RetrieveAssetValueResult{AssetValue: &model.AssetValue{
				ID:         1,
				AssetID:    1,
				Value:      decimal.NewFromInt(123),
				ObservedAt: now,
			}},
			wantErr: false,
		},
	}

	for _, tc := range tt {
		s.T().Run(tc.name, func(t *testing.T) {
			env := s.NewTestWorkflowEnvironment()
			tc.setupEnv(env)

			env.ExecuteWorkflow(RetrieveAssetValue, tc.params)

			err := env.GetWorkflowError()

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				var result *avt.RetrieveAssetValueResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				assert.Equal(t, tc.result, result)
			}
		})
	}
}
