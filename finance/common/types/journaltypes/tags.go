package journaltypes

import (
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
)

// SetJournalEntryTagsInput replaces the tags of an entry of any status, and
// none clears them.
type SetJournalEntryTagsInput struct {
	JournalEntryID int64   `json:"journalEntryId"`
	TagIDs         []int64 `json:"tagIds"`
}

// SetJournalEntryTagsResult carries the tags of the entry afterwards, each once
// and in id order.
type SetJournalEntryTagsResult struct {
	TagIDs []int64 `json:"tagIds"`
}

// Validate checks the command without the database.
func (i *SetJournalEntryTagsInput) Validate() error {
	if i.JournalEntryID <= 0 {
		return internalerrors.ErrInvalidParameter{Description: "journalEntryId must be greater than 0"}
	}
	return ValidateTagIDs(i.TagIDs)
}
