package tag

import (
	"testing"
	"time"

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
		"nominal by Name in another casing": {
			params: tt.DeleteTagParams{Name: "TEST TAG"},
		},
		"nominal by Name with surrounding space": {
			params: tt.DeleteTagParams{Name: "  test tag  "},
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
					query.Where("lower(name) = lower(trim(?))", tc.params.Name)
				}
				exists, errExists = query.Exists(s.T().Context())
				require.NoError(t, errExists)
				require.False(t, exists)
			}
		})
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, &tag, id)
	}
}

// Test_DeleteTag_Assigned checks that deleting a tag an entry has detaches it
// from the entry and keeps the entry.
func (s *UnitTestSuite) Test_DeleteTag_Assigned() {
	entry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: model.JournalEntryDraft}
	tag := &model.Tag{Name: "vacation"}
	temporaltesting.Insert(s.T(), s.DB, entry)
	temporaltesting.Insert(s.T(), s.DB, tag)
	_, err := s.DB.NewInsert().Model(&model.JournalEntryTag{JournalEntryID: entry.ID, TagID: tag.ID}).Exec(s.T().Context())
	require.NoError(s.T(), err)
	defer temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.JournalEntry)(nil), entry.ID)

	_, err = s.Env.ExecuteActivity(s.activity.DeleteTag, tt.DeleteTagParams{ID: tag.ID})
	require.NoError(s.T(), err)

	require.Zero(s.T(), s.CountWhere[model.Tag]("id = ?", tag.ID))
	require.Zero(s.T(), s.Count[model.JournalEntryTag]())
	require.Equal(s.T(), 1, s.CountWhere[model.JournalEntry]("id = ?", entry.ID))
}
