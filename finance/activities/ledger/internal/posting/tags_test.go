package posting

import (
	"context"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// tagIDsOf returns the tags of the entry in id order.
func (s *UnitTestSuite) tagIDsOf(entryID int64) []int64 {
	var links []model.JournalEntryTag
	require.NoError(s.T(), s.DB.NewSelect().Model(&links).Where("journal_entry_id = ?", entryID).Order("tag_id").Scan(s.T().Context()))
	ids := make([]int64, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.TagID)
	}
	return ids
}

// Test_PostEntryTags posts an entry with tags. The cases name the tags by
// index into two tags created for the case, and -1 for one that does not
// exist.
func (s *UnitTestSuite) Test_PostEntryTags() {
	tests := []struct {
		name    string
		tags    []int
		want    []int
		wantErr string
	}{
		{name: "no tags"},
		{name: "two tags", tags: []int{1, 0}, want: []int{0, 1}},
		{name: "a tag given twice is taken once", tags: []int{0, 1, 0}, want: []int{0, 1}},
		{name: "a tag that does not exist fails the post", tags: []int{0, -1}, wantErr: "Tag not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := s.T().Context()
			tags := []*model.Tag{{Name: "a " + tt.name}, {Name: "b " + tt.name}}
			for _, tag := range tags {
				temporaltesting.Insert(s.T(), s.DB, tag)
			}
			idOf := func(i int) int64 {
				if i < 0 {
					return 999999
				}
				return tags[i].ID
			}

			input := s.entry()
			input.IdempotencyKey = tt.name
			for _, i := range tt.tags {
				input.TagIDs = append(input.TagIDs, idOf(i))
			}

			var entryID int64
			err := s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) (err error) {
				entryID, err = PostJournalEntryTx(ctx, tx, input)
				return err
			})

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				require.Zero(s.T(), s.CountWhere[model.JournalEntry]("idempotency_key = ?", tt.name), "the entry goes with its tags")
			} else {
				require.NoError(s.T(), err)
				want := []int64{}
				for _, i := range tt.want {
					want = append(want, idOf(i))
				}
				require.Equal(s.T(), want, s.tagIDsOf(entryID))
			}
			for _, tag := range tags {
				temporaltesting.Delete(s.T(), s.DB, (*model.Tag)(nil), tag.ID)
			}
		})
	}
}

// Test_RewriteEntryTx_Tags rewrites an entry tagged with the first of three
// tags. The cases name the tags by index, and -1 for one that does not exist.
func (s *UnitTestSuite) Test_RewriteEntryTx_Tags() {
	tests := []struct {
		name    string
		tags    []int
		want    []int
		wantErr string
	}{
		{name: "the tags of the input replace the entry's", tags: []int{1, 2}, want: []int{1, 2}},
		{name: "the entry keeps a tag the input repeats", tags: []int{0, 2, 2}, want: []int{0, 2}},
		{name: "no tags clear the entry's"},
		{name: "a tag that does not exist keeps the entry's", tags: []int{1, -1}, want: []int{0}, wantErr: "Tag not found"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := s.T().Context()
			tags := []*model.Tag{{Name: "a " + tt.name}, {Name: "b " + tt.name}, {Name: "c " + tt.name}}
			for _, tag := range tags {
				temporaltesting.Insert(s.T(), s.DB, tag)
			}
			idOf := func(i int) int64 {
				if i < 0 {
					return 999999
				}
				return tags[i].ID
			}

			entry := &model.JournalEntry{OccurredAt: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)}
			temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, entry,
				model.Posting{LedgerAccountID: s.debit.ID, Amount: decimal.NewFromInt(7), Side: model.PostingDebit},
				model.Posting{LedgerAccountID: s.credit.ID, Amount: decimal.NewFromInt(7), Side: model.PostingCredit},
			)
			_, err := s.DB.NewInsert().Model(&model.JournalEntryTag{JournalEntryID: entry.ID, TagID: tags[0].ID}).Exec(ctx)
			require.NoError(s.T(), err)

			input := lt.PostJournalEntryInput{
				OccurredAt: entry.OccurredAt,
				Postings: []lt.PostingInput{
					{LedgerAccountID: s.debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
					{LedgerAccountID: s.credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
				},
			}
			for _, i := range tt.tags {
				input.TagIDs = append(input.TagIDs, idOf(i))
			}

			err = s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				var locked model.JournalEntry
				if err := tx.NewSelect().Model(&locked).Where("je.id = ?", entry.ID).For("UPDATE").Scan(ctx); err != nil {
					return err
				}
				return RewriteEntryTx(ctx, tx, &locked, input, nil, []int64{s.debit.ID, s.credit.ID})
			})
			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
			} else {
				require.NoError(s.T(), err)
			}

			want := []int64{}
			for _, i := range tt.want {
				want = append(want, idOf(i))
			}
			require.Equal(s.T(), want, s.tagIDsOf(entry.ID))
			for _, tag := range tags {
				temporaltesting.Delete(s.T(), s.DB, (*model.Tag)(nil), tag.ID)
			}
		})
	}
}
