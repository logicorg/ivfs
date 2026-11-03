package tag

import (
	"testing"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_DeleteTag() {
	tag := model.Tag{
		Name: "test tag",
	}

	tcs := map[string]struct {
		params tt.DeleteTagParams
		expErr bool
	}{
		"nominal by ID": {
			params: tt.DeleteTagParams{ID: 1},
		},
		"nominal by Name": {
			params: tt.DeleteTagParams{Name: tag.Name},
		},
		"should succeed not found": {
			params: tt.DeleteTagParams{ID: 999},
		},
		"error invalid id": {
			params: tt.DeleteTagParams{ID: 0},
			expErr: true,
		},
		"non-existing id": {
			params: tt.DeleteTagParams{ID: 999},
		},
		"error no params": {
			params: tt.DeleteTagParams{},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		temporaltesting.Insert(s.T(), s.DB, &tag)
		id := tag.ID
		s.T().Run(s.T().Name()+"/"+name, func(t *testing.T) {
			_, err := s.Env.ExecuteActivity(s.activity.DeleteTag, tc.params)

			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if !tc.expErr {
				var exists bool
				var errExists error
				query := s.DB.NewSelect().Model((*model.Tag)(nil))
				if tc.params.ID > 0 {
					query.Where("id = ?", tc.params.ID)
				} else if tc.params.Name != "" {
					query.Where("name = ?", tc.params.Name)
				}
				exists, errExists = query.Exists(s.T().Context())
				require.NoError(t, errExists)
				require.False(t, exists)
			}
		})
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, &tag, id)
	}
}
