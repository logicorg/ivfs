package tag

import (
	"testing"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_InsertTag() {
	tcs := map[string]struct {
		params  tt.InsertTagParams
		expName string
		expErr  bool
	}{
		"nominal": {
			params: tt.InsertTagParams{
				Name:        "test tag",
				Description: ptr.NewPtr("test description"),
			},
			expName: "test tag",
		},
		"nominal - name is trimmed": {
			params: tt.InsertTagParams{
				Name:        "  test tag  ",
				Description: ptr.NewPtr("test description"),
			},
			expName: "test tag",
		},
		"error - no name": {
			params: tt.InsertTagParams{
				Description: ptr.NewPtr("test description"),
			},
			expErr: true,
		},
		"error - blank name": {
			params: tt.InsertTagParams{Name: "   "},
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

			var inserted model.Tag
			temporaltesting.Retrieve(t, s.DB, &inserted, res.ID)
			require.Equal(t, tc.expName, inserted.Name)
			require.Equal(t, *tc.params.Description, inserted.Description)

			temporaltesting.Delete(t, s.DB, &inserted, res.ID)
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

// Test_InsertTag_ExistingName checks that a name matching a tag in another
// casing or with surrounding space finds that tag and leaves it unchanged.
func (s *UnitTestSuite) Test_InsertTag_ExistingName() {
	tcs := map[string]string{
		"another casing":    "TRAVEL",
		"surrounding space": "  Travel  ",
	}

	for name, input := range tcs {
		existing := &model.Tag{Name: "Travel", Description: "kept"}
		temporaltesting.Insert(s.T(), s.DB, existing)
		s.T().Run(name, func(t *testing.T) {
			actRes, err := s.Env.ExecuteActivity(s.activity.InsertTag, tt.InsertTagParams{Name: input, Description: ptr.NewPtr("dropped")})
			require.NoError(t, err)

			var res tt.InsertTagResult
			require.NoError(t, actRes.Get(&res))
			require.Equal(t, existing.ID, res.ID)

			var stored model.Tag
			temporaltesting.Retrieve(t, s.DB, &stored, existing.ID)
			require.Equal(t, "Travel", stored.Name)
			require.Equal(t, "kept", stored.Description)
		})
		temporaltesting.Delete(s.T(), s.DB, existing, existing.ID)
	}
}
