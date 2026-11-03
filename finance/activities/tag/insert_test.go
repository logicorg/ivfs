package tag

import (
	"testing"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_InsertTag() {
	tcs := map[string]struct {
		params tt.InsertTagParams
		expErr bool
	}{
		"nominal": {
			params: tt.InsertTagParams{
				Name:        "test tag",
				Description: ptr.NewPtr("test description"),
			},
		},
		"error - no name": {
			params: tt.InsertTagParams{
				Description: ptr.NewPtr("test description"),
			},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			actRes, err := s.Env.ExecuteActivity(s.activity.InsertTag, tc.params)

			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			var res tt.InsertTagResult
			err = actRes.Get(&res)
			require.NoError(t, err)
			require.NotZero(t, res.ID)
		})
	}
}

func (s *UnitTestSuite) Test_InsertTag_Idempotency() {
	insertParams := tt.InsertTagParams{
		Name: "test tag",
	}

	// insert
	res, err := s.Env.ExecuteActivity(s.activity.InsertTag, insertParams)
	require.NoError(s.T(), err)

	insertResult := &tt.InsertTagResult{}
	require.NoError(s.T(), res.Get(insertResult))
	require.NotZero(s.T(), insertResult.ID)

	// insert again
	res, err = s.Env.ExecuteActivity(s.activity.InsertTag, insertParams)
	require.NoError(s.T(), err)

	secondInsertResult := &tt.InsertTagResult{}
	require.NoError(s.T(), res.Get(secondInsertResult))
	require.Equal(s.T(), insertResult, secondInsertResult)
}
