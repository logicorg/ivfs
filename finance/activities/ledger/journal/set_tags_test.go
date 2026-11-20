package journal

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

// tagsOf returns the tags of the entry in id order.
func (s *UnitTestSuite) tagsOf(entryID int64) []int64 {
	var links []model.JournalEntryTag
	require.NoError(s.T(), s.DB.NewSelect().Model(&links).Where("journal_entry_id = ?", entryID).Order("tag_id").Scan(s.T().Context()))
	ids := []int64{}
	for _, link := range links {
		ids = append(ids, link.TagID)
	}
	return ids
}

// Test_SetJournalEntryTags sets the tags of a draft entry tagged with the
// first of three tags. The cases name the tags by index, and -1 for one that
// does not exist.
func (s *UnitTestSuite) Test_SetJournalEntryTags() {
	tests := []struct {
		name string
		tags []int
		// unknownEntry sets the tags of an entry that does not exist.
		unknownEntry bool
		// zeroID adds a tag id of zero, which no tag has.
		zeroID  bool
		want    []int
		wantErr string
	}{
		{name: "the tags of the input replace the entry's", tags: []int{2, 1}, want: []int{1, 2}},
		{name: "a tag given twice is taken once", tags: []int{0, 2, 0}, want: []int{0, 2}},
		{name: "no tags clear the entry's"},
		{name: "a tag that does not exist keeps the entry's", tags: []int{1, -1}, want: []int{0}, wantErr: "Tag not found"},
		{name: "an entry that does not exist", tags: []int{1}, unknownEntry: true, want: []int{0}, wantErr: "JournalEntry not found"},
		{name: "a tag id of zero", tags: []int{1}, zeroID: true, want: []int{0}, wantErr: "tag id 0"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := s.T().Context()
			entry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: model.JournalEntryDraft}
			temporaltesting.Insert(s.T(), s.DB, entry)
			tags := []*model.Tag{{Name: "a " + tt.name}, {Name: "b " + tt.name}, {Name: "c " + tt.name}}
			for _, tag := range tags {
				temporaltesting.Insert(s.T(), s.DB, tag)
			}
			_, err := s.DB.NewInsert().Model(&model.JournalEntryTag{JournalEntryID: entry.ID, TagID: tags[0].ID}).Exec(ctx)
			require.NoError(s.T(), err)
			idOf := func(i int) int64 {
				if i < 0 {
					return 999999
				}
				return tags[i].ID
			}

			input := lt.SetJournalEntryTagsInput{JournalEntryID: entry.ID}
			if tt.unknownEntry {
				input.JournalEntryID = 999999
			}
			for _, i := range tt.tags {
				input.TagIDs = append(input.TagIDs, idOf(i))
			}
			if tt.zeroID {
				input.TagIDs = append(input.TagIDs, 0)
			}

			want := []int64{}
			for _, i := range tt.want {
				want = append(want, idOf(i))
			}

			got, err := s.Env.ExecuteActivity(s.ja.SetJournalEntryTags, input)
			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
			} else {
				require.NoError(s.T(), err)
				var result lt.SetJournalEntryTagsResult
				require.NoError(s.T(), got.Get(&result))
				require.Equal(s.T(), want, append([]int64{}, result.TagIDs...))
			}
			require.Equal(s.T(), want, s.tagsOf(entry.ID))

			temporaltesting.Delete(s.T(), s.DB, (*model.JournalEntry)(nil), entry.ID)
			for _, tag := range tags {
				temporaltesting.Delete(s.T(), s.DB, (*model.Tag)(nil), tag.ID)
			}
		})
	}
}

// Test_SetJournalEntryTags_Idempotent checks that setting the same tags again
// leaves the same tags and returns the same result.
func (s *UnitTestSuite) Test_SetJournalEntryTags_Idempotent() {
	entry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, entry)
	a, b := &model.Tag{Name: "vacation"}, &model.Tag{Name: "reimbursable"}
	temporaltesting.Insert(s.T(), s.DB, a)
	temporaltesting.Insert(s.T(), s.DB, b)
	input := lt.SetJournalEntryTagsInput{JournalEntryID: entry.ID, TagIDs: []int64{b.ID, a.ID}}

	var results [2]lt.SetJournalEntryTagsResult
	for i := range results {
		got, err := s.Env.ExecuteActivity(s.ja.SetJournalEntryTags, input)
		require.NoError(s.T(), err)
		require.NoError(s.T(), got.Get(&results[i]))
	}

	require.Equal(s.T(), results[0], results[1])
	require.Equal(s.T(), []int64{a.ID, b.ID}, s.tagsOf(entry.ID))
	require.Equal(s.T(), 2, s.Count[model.JournalEntryTag]())
}
