package sqllib_test

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// Test_JournalEntryTags checks the rules of the tags of an entry:
// a tag is assigned once and only if it exists, and deleting the entry or the
// tag deletes the assignment and nothing else. Each case starts from a draft
// entry, which needs no postings, and two tags.
//
// The statements name their rows with :entry, :a and :b (the two tags).
func (s *UnitTestSuite) Test_JournalEntryTags() {
	tests := []struct {
		name       string
		statements []string
		wantErr    string
		// wantAssigned is the number of assignments left after the statements.
		wantAssigned int
		// wantEntry says whether the entry is still there.
		wantEntry bool
	}{
		{
			name: "an entry takes two tags",
			statements: []string{
				`INSERT INTO ledger.journal_entry_tag (journal_entry_id, tag_id) VALUES (:entry, :a), (:entry, :b)`,
			},
			wantAssigned: 2,
			wantEntry:    true,
		},
		{
			name: "an entry takes the same tag twice",
			statements: []string{
				`INSERT INTO ledger.journal_entry_tag (journal_entry_id, tag_id) VALUES (:entry, :a), (:entry, :a)`,
			},
			wantErr: "journal_entry_tag_pkey",
		},
		{
			name: "an entry takes a tag that does not exist",
			statements: []string{
				`INSERT INTO ledger.journal_entry_tag (journal_entry_id, tag_id) VALUES (:entry, 999999)`,
			},
			wantErr: "journal_entry_tag_tag_id_fkey",
		},
		{
			name: "deleting the entry deletes its assignments",
			statements: []string{
				`INSERT INTO ledger.journal_entry_tag (journal_entry_id, tag_id) VALUES (:entry, :a), (:entry, :b)`,
				`DELETE FROM ledger.journal_entry WHERE id = :entry`,
			},
		},
		{
			name: "deleting a tag detaches it and keeps the entry's other tags",
			statements: []string{
				`INSERT INTO ledger.journal_entry_tag (journal_entry_id, tag_id) VALUES (:entry, :a), (:entry, :b)`,
				`DELETE FROM tag.tag WHERE id = :b`,
			},
			wantAssigned: 1,
			wantEntry:    true,
		},
	}

	for _, tt := range tests {
		entry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: model.JournalEntryDraft}
		a := &model.Tag{Name: "vacation"}
		b := &model.Tag{Name: "reimbursable"}
		temporaltesting.Insert(s.T(), s.DB, entry)
		temporaltesting.Insert(s.T(), s.DB, a)
		temporaltesting.Insert(s.T(), s.DB, b)

		s.Run(tt.name, func() {
			names := strings.NewReplacer(
				":entry", strconv.FormatInt(entry.ID, 10),
				":a", strconv.FormatInt(a.ID, 10),
				":b", strconv.FormatInt(b.ID, 10),
			)
			err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
				for _, statement := range tt.statements {
					if _, err := tx.ExecContext(ctx, names.Replace(statement)); err != nil {
						return err
					}
				}
				return nil
			})
			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				return
			}
			require.NoError(s.T(), err)

			require.Equal(s.T(), tt.wantAssigned, s.Count[model.JournalEntryTag]())
			wantEntries := 0
			if tt.wantEntry {
				wantEntries = 1
			}
			require.Equal(s.T(), wantEntries, s.CountWhere[model.JournalEntry]("id = ?", entry.ID))
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.JournalEntry)(nil), entry.ID)
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Tag)(nil), a.ID)
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Tag)(nil), b.ID)
	}
}
