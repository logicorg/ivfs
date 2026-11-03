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

type UpdateWorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestUpdateWorkflowUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateWorkflowUnitTestSuite))
}

func (s *UpdateWorkflowUnitTestSuite) TestUpdate() {
	now := time.Now().UTC()
	updatedValue := decimal.NewFromInt(200)

	tt := []struct {
		name     string
		params   avt.UpdateAssetValueParams
		setupEnv func(env *testsuite.TestWorkflowEnvironment)
		result   *avt.UpdateAssetValueResult
		wantErr  bool
	}{
		{
			name:   "should fail with invalid parameters",
			params: avt.UpdateAssetValueParams{},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				ava := value.AssetValueActivity{}
				env.OnActivity(ava.UpdateAssetValue, mock.Anything, avt.UpdateAssetValueParams{}).
					Return(nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ID"}})
			},
			result:  nil,
			wantErr: true,
		},
		{
			name: "should succeed with valid params",
			params: avt.UpdateAssetValueParams{
				ID:    1,
				Value: &updatedValue,
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				ava := value.AssetValueActivity{}
				env.OnActivity(ava.UpdateAssetValue, mock.Anything, avt.UpdateAssetValueParams{
					ID:    1,
					Value: &updatedValue,
				}).
					Return(&avt.UpdateAssetValueResult{AssetValue: &model.AssetValue{
						ID:         1,
						AssetID:    1,
						Value:      updatedValue,
						ObservedAt: now,
					}}, nil)
			},
			result: &avt.UpdateAssetValueResult{AssetValue: &model.AssetValue{
				ID:         1,
				AssetID:    1,
				Value:      updatedValue,
				ObservedAt: now,
			}},
			wantErr: false,
		},
	}

	for _, tc := range tt {
		s.T().Run(tc.name, func(t *testing.T) {
			env := s.NewTestWorkflowEnvironment()
			tc.setupEnv(env)

			env.ExecuteWorkflow(UpdateAssetValue, tc.params)

			err := env.GetWorkflowError()

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				var result *avt.UpdateAssetValueResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				assert.Equal(t, tc.result, result)
			}
		})
	}
}
