package categories

import (
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_CloseCategory() {
	at := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		// category is the fixture's name for the category to close.
		category string
		// at is when to close it, and now when zero.
		at time.Time
		// wantClosedAt is when the category is closed afterwards; zero means
		// around now.
		wantClosedAt time.Time
		wantErr      string
	}{
		{name: "an open category, at a given time", category: "travel", at: at, wantClosedAt: at},
		{name: "an open category, now", category: "travel"},
		{name: "a category with entries, which stay", category: "rent", at: at, wantClosedAt: at},
		{name: "a child", category: "citrus", at: at, wantClosedAt: at},
		{name: "a parent whose children are closed", category: "attic", at: at, wantClosedAt: at},
		{name: "a closed category keeps its time", category: "old", at: at, wantClosedAt: closedAt},
		{name: "a parent with open children", category: "food", at: at, wantErr: "has open categories under it"},
		{name: "a category that does not exist", category: "missing", at: at, wantErr: "Category not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := s.seed()
			defer s.clear()
			id := *f.id(tt.category)
			postings := s.CountWhere[model.Posting]("category_id = ?", id)
			started := time.Now().UTC().Add(-time.Second)

			_, err := s.Env.ExecuteActivity(s.ca.CloseCategory, ct.CloseCategoryInput{CategoryID: id, ClosedAt: tt.at})

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				if category := s.stored(id); category != nil {
					require.Nil(s.T(), category.ClosedAt)
				}
				return
			}
			require.NoError(s.T(), err)

			closed := s.stored(id)
			require.NotNil(s.T(), closed.ClosedAt)
			if tt.wantClosedAt.IsZero() {
				require.True(s.T(), closed.ClosedAt.After(started), "closed at %s", closed.ClosedAt)
			} else {
				require.True(s.T(), tt.wantClosedAt.Equal(*closed.ClosedAt), "closed at %s", closed.ClosedAt)
			}
			require.Equal(s.T(), postings, s.CountWhere[model.Posting]("category_id = ?", id))
		})
	}
}

func (s *UnitTestSuite) Test_CloseCategory_IsIdempotent() {
	f := s.seed()
	first := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	for _, at := range []time.Time{first, first.AddDate(0, 1, 0)} {
		_, err := s.Env.ExecuteActivity(s.ca.CloseCategory, ct.CloseCategoryInput{CategoryID: f["travel"].ID, ClosedAt: at})
		require.NoError(s.T(), err)
		require.True(s.T(), first.Equal(*s.stored(f["travel"].ID).ClosedAt), "the first close is the one that counts")
	}
}
