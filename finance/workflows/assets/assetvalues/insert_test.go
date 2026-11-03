package assetvalues

import (
	"reflect"
	"testing"
	"time"

	"github.com/logicorg/ivfs/finance/activities/asset/value"
	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
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
	now := time.Now()
	tt := []struct {
		name     string
		params   avt.InsertAssetValueParams
		setupEnv func(env *testsuite.TestWorkflowEnvironment)
		result   *avt.InsertAssetValueResult
		wantErr  bool
	}{
		{
			name:   "should fail with invalid parameters",
			params: avt.InsertAssetValueParams{},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				ava := value.AssetValueActivity{}
				env.OnActivity(ava.InsertAssetValue, mock.AnythingOfType(reflect.TypeOf(avt.InsertAssetValueParams{}).String())).
					Return(nil, errors.ErrMissingRequiredParameter{Parameters: []string{"AssetID"}})
			},
			result:  nil,
			wantErr: true,
		},
		{
			name: "should succeed with valid params",
			params: avt.InsertAssetValueParams{
				AssetID:    1,
				CurrencyID: 1,
				Value:      decimal.NewFromInt(100),
				Date:       now,
			},
			setupEnv: func(env *testsuite.TestWorkflowEnvironment) {
				ava := value.AssetValueActivity{}
				env.OnActivity(ava.InsertAssetValue, mock.Anything, mock.AnythingOfType(reflect.TypeOf(avt.InsertAssetValueParams{}).String())).Return(&avt.InsertAssetValueResult{ID: 1}, nil)
			},
			result:  &avt.InsertAssetValueResult{ID: 1},
			wantErr: false,
		},
	}

	for _, tc := range tt {
		s.T().Run(tc.name, func(t *testing.T) {
			env := s.NewTestWorkflowEnvironment()
			tc.setupEnv(env)

			env.ExecuteWorkflow(InsertAssetValue, tc.params)

			err := env.GetWorkflowError()

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				var result *avt.InsertAssetValueResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				assert.Equal(t, tc.result, result)
			}
		})
	}
}
