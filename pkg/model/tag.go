package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Tag represents the tag.tag table.
type Tag struct {
	bun.BaseModel `bun:"table:tag.tag,alias:tg"`

	ID          int64     `bun:"id,pk,autoincrement" json:"id" jsonschema:"id of the tag"`
	Name        string    `bun:"name,notnull,unique" json:"name" jsonschema:"name of the tag"`
	Description string    `bun:"description" json:"description" jsonschema:"description of the tag"`
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt" jsonschema:"timestamp when the tag was created"`
	UpdatedAt   time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt" jsonschema:"timestamp when the tag was last updated"`
}

// ToString returns the tag as a string
func (t Tag) ToString() string {
	return t.Name
}

// FindTagByNameQuery returns a query to retrieve a Tag by name.
func FindTagByNameQuery(db *bun.DB, name string) *bun.SelectQuery {
	return db.NewSelect().Model((*Tag)(nil)).Where("tg.name = ?", name)
}

// FindTagByIDQuery returns a query to find a Tag by ID.
func FindTagByIDQuery(db *bun.DB, id int64) *bun.SelectQuery {
	return db.NewSelect().Model((*Tag)(nil)).Where("tg.id = ?", id)
}
