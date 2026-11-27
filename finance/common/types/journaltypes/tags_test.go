package journaltypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetJournalEntryTagsInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   SetJournalEntryTagsInput
		wantErr string
	}{
		{name: "tags", input: SetJournalEntryTagsInput{JournalEntryID: 1, TagIDs: []int64{2, 3}}},
		{name: "no tags", input: SetJournalEntryTagsInput{JournalEntryID: 1}},
		{name: "no entry", input: SetJournalEntryTagsInput{TagIDs: []int64{2}}, wantErr: "journalEntryId"},
		{name: "a tag id of zero", input: SetJournalEntryTagsInput{JournalEntryID: 1, TagIDs: []int64{2, 0}}, wantErr: "tag id 0"},
		{name: "a negative tag id", input: SetJournalEntryTagsInput{JournalEntryID: 1, TagIDs: []int64{-2}}, wantErr: "tag id -2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
