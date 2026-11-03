package tag

import (
	"testing"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_UpdateTag() {
	tag := &model.Tag{
		Name: "test tag",
	}

	tcs := map[string]struct {
		params   tt.UpdateTagParams
		expErr   bool
		expected model.Tag
	}{
		"nominal": {
			params: tt.UpdateTagParams{
				Name: ptr.NewPtr("new name"),
				ID:   1,
			},
			expected: model.Tag{
				Name: "new name",
			},
		},
		"nominal - update description": {
			params: tt.UpdateTagParams{
				Description: ptr.NewPtr("new description"),
				ID:          1,
			},
			expected: model.Tag{
				Name:        "test tag",
				Description: "new description",
			},
		},
		"error - not found": {
			params: tt.UpdateTagParams{
				Name: ptr.NewPtr("new name"),
				ID:   999,
			},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		temporaltesting.Insert(s.T(), s.DB, tag)
		id := tag.ID
		s.T().Run(name, func(t *testing.T) {
			_, err := s.Env.ExecuteActivity(s.activity.UpdateTag, tc.params)

			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// check that the tag is updated
			var updatedTag model.Tag
			err = s.DB.NewSelect().Model(&updatedTag).Where("id = ?", tc.params.ID).Scan(s.T().Context())
			require.NoError(t, err)
		})
		temporaltesting.Delete(s.T(), s.DB, tag, id)
	}
}
