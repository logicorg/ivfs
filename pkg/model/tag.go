package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Tag represents the tag.tag table.
type Tag struct {
	bun.BaseModel `bun:"table:tag.tag,alias:tg"`

	ID          int64     `bun:"id,pk,autoincrement" json:"id" jsonschema:"id of the tag"`
	Name        string    `bun:"name,notnull" json:"name" jsonschema:"name of the tag"`
	Description string    `bun:"description" json:"description" jsonschema:"description of the tag"`
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt" jsonschema:"timestamp when the tag was created"`
	UpdatedAt   time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt" jsonschema:"timestamp when the tag was last updated"`
}

// JournalEntryTag assigns a tag to a journal entry. Deleting either
// side deletes the assignment.
type JournalEntryTag struct {
	bun.BaseModel `bun:"table:ledger.journal_entry_tag,alias:jet"`

	JournalEntryID int64 `bun:"journal_entry_id,pk" json:"journalEntryId"`
	TagID          int64 `bun:"tag_id,pk" json:"tagId"`

	JournalEntry *JournalEntry `bun:"rel:belongs-to,join:journal_entry_id=id" json:"journalEntry,omitempty"`
	Tag          *Tag          `bun:"rel:belongs-to,join:tag_id=id" json:"tag,omitempty"`
}

// ToString returns the tag as a string
func (t Tag) ToString() string {
	return t.Name
}

// FindTagByNameQuery returns a query to retrieve a Tag by name, compared
// case-insensitively (ADR 0005).
func FindTagByNameQuery(db *bun.DB, name string) *bun.SelectQuery {
	return db.NewSelect().Model((*Tag)(nil)).Where("lower(tg.name) = lower(?)", name)
}

// FindTagByIDQuery returns a query to find a Tag by ID.
func FindTagByIDQuery(db *bun.DB, id int64) *bun.SelectQuery {
	return db.NewSelect().Model((*Tag)(nil)).Where("tg.id = ?", id)
}
