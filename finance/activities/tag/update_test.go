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
	tag := &model.Tag{ID: 1, Name: "test tag", Description: "test description"}
	taken := &model.Tag{ID: 2, Name: "Taken"}

	tcs := map[string]struct {
		params   tt.UpdateTagParams
		expErr   string
		expected model.Tag
	}{
		"nominal": {
			params:   tt.UpdateTagParams{ID: 1, Name: ptr.NewPtr("new name")},
			expected: model.Tag{Name: "new name", Description: "test description"},
		},
		"nominal - update description": {
			params:   tt.UpdateTagParams{ID: 1, Description: ptr.NewPtr("new description")},
			expected: model.Tag{Name: "test tag", Description: "new description"},
		},
		"nominal - clear description": {
			params:   tt.UpdateTagParams{ID: 1, Description: ptr.NewPtr("")},
			expected: model.Tag{Name: "test tag"},
		},
		"nominal - name is trimmed": {
			params:   tt.UpdateTagParams{ID: 1, Name: ptr.NewPtr("  new name  ")},
			expected: model.Tag{Name: "new name", Description: "test description"},
		},
		"nominal - own name in another casing": {
			params:   tt.UpdateTagParams{ID: 1, Name: ptr.NewPtr("TEST TAG")},
			expected: model.Tag{Name: "TEST TAG", Description: "test description"},
		},
		"error - not found": {
			params: tt.UpdateTagParams{ID: 999, Name: ptr.NewPtr("new name")},
			expErr: "resource Tag not found",
		},
		"error - blank name": {
			params: tt.UpdateTagParams{ID: 1, Name: ptr.NewPtr("   ")},
			expErr: "name must not be blank",
		},
		"error - name taken in another casing": {
			params: tt.UpdateTagParams{ID: 1, Name: ptr.NewPtr("taken")},
			expErr: "resource Tag conflict",
		},
	}

	for name, tc := range tcs {
		temporaltesting.Insert(s.T(), s.DB, tag)
		temporaltesting.Insert(s.T(), s.DB, taken)
		s.T().Run(name, func(t *testing.T) {
			_, err := s.Env.ExecuteActivity(s.activity.UpdateTag, tc.params)

			if tc.expErr != "" {
				require.ErrorContains(t, err, tc.expErr)
				return
			}

			require.NoError(t, err)

			var updatedTag model.Tag
			err = s.DB.NewSelect().Model(&updatedTag).Where("id = ?", tc.params.ID).Scan(s.T().Context())
			require.NoError(t, err)
			require.Equal(t, tc.expected.Name, updatedTag.Name)
			require.Equal(t, tc.expected.Description, updatedTag.Description)
		})
		temporaltesting.Delete(s.T(), s.DB, tag, tag.ID)
		temporaltesting.Delete(s.T(), s.DB, taken, taken.ID)
	}
}

// Test_UpdateTag_Idempotency checks that a retried update leaves the tag as
// the first run did.
func (s *UnitTestSuite) Test_UpdateTag_Idempotency() {
	tag := &model.Tag{Name: "test tag"}
	temporaltesting.Insert(s.T(), s.DB, tag)
	params := tt.UpdateTagParams{ID: tag.ID, Name: ptr.NewPtr("new name"), Description: ptr.NewPtr("new description")}

	for range 2 {
		_, err := s.Env.ExecuteActivity(s.activity.UpdateTag, params)
		require.NoError(s.T(), err)
	}

	var updatedTag model.Tag
	temporaltesting.Retrieve(s.T(), s.DB, &updatedTag, tag.ID)
	require.Equal(s.T(), "new name", updatedTag.Name)
	require.Equal(s.T(), "new description", updatedTag.Description)
}
