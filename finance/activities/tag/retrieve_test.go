package tag

import (
	"testing"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_RetrieveTag() {
	// setup
	tag := &model.Tag{
		Name: "test tag",
	}
	temporaltesting.Insert(s.T(), s.DB, tag)

	tcs := map[string]struct {
		params  tt.RetrieveTagParams
		expErr  bool
		expResp *model.Tag
	}{
		"by id": {
			params: tt.RetrieveTagParams{
				ID: tag.ID,
			},
			expResp: tag,
		},
		"by name": {
			params: tt.RetrieveTagParams{
				Name: tag.Name,
			},
			expResp: tag,
		},
		"by name in another casing": {
			params: tt.RetrieveTagParams{
				Name: "TEST TAG",
			},
			expResp: tag,
		},
		"by name with surrounding space": {
			params: tt.RetrieveTagParams{
				Name: "  test tag  ",
			},
			expResp: tag,
		},
		"not found": {
			params: tt.RetrieveTagParams{
				ID: 999,
			},
			expErr: true,
		},
		"error - no id or name": {
			params: tt.RetrieveTagParams{},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			resp, err := s.Env.ExecuteActivity(s.activity.RetrieveTag, tc.params)

			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			var res tt.RetrieveTagResult
			err = resp.Get(&res)
			require.NoError(t, err)

			require.Equal(t, tc.expResp.Name, res.Tag.Name)
		})
	}
}
