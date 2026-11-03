package model

import (
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// Institution is who a financial account is held with. Its name is unique
// compared case-insensitively, through an index on lower(name) rather than a
// column constraint (ADR 0005).
type Institution struct {
	bun.BaseModel `bun:"table:institution.institution,alias:i"`

	ID   int64  `bun:"id,pk,autoincrement" json:"id" jsonschema:"id of the institution"`
	Name string `bun:"name,notnull" json:"name" jsonschema:"name of the institution"`
	Kind string `bun:"kind,notnull" json:"kind" jsonschema:"kind of the institution"`

	OpenedAt *time.Time `bun:"opened_at,nullzero" json:"openedAt,omitempty" jsonschema:"timestamp when the institution was opened, if applicable"`
	ClosedAt *time.Time `bun:"closed_at,nullzero" json:"closedAt,omitempty" jsonschema:"timestamp when the institution was closed, if applicable"`

	// CreatedAt is the timestamp when the institution was created in the database.
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt" jsonschema:"timestamp when the institution was created"`
	// UpdatedAt is the timestamp when the institution was last updated.
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt" jsonschema:"timestamp when the institution was last updated"`

	// FinancialAccounts are the relationships held with this institution.
	FinancialAccounts []FinancialAccount `bun:"rel:has-many,join:id=institution_id" json:"financialAccounts,omitempty"`
}

// FindInstitutionByIDQuery returns an institution by ID.
func FindInstitutionByIDQuery(db bun.IDB, id int64) *bun.SelectQuery {
	return db.NewSelect().Model((*Institution)(nil)).Where("i.id = ?", id)
}

// FindInstitutionByNameQuery returns the institution whose name matches name,
// ignoring case and surrounding space.
func FindInstitutionByNameQuery(db bun.IDB, name string) *bun.SelectQuery {
	return db.NewSelect().Model((*Institution)(nil)).Where("lower(i.name) = lower(?)", strings.TrimSpace(name))
}
