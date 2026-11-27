package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Category classifies an income or an expense, in any currency. It has no
// balance and no currency of its own: a posting to an income or expense ledger
// account may carry one, and the category's total is the sum of those
// postings. Categories nest at most two parent links deep, and a child has its
// parent's kind. The name is unique among the siblings of a kind, compared
// case-insensitively through an index on lower(name).
type Category struct {
	bun.BaseModel `bun:"table:ledger.category,alias:cat"`

	ID       int64  `bun:"id,pk,autoincrement" json:"id"`
	ParentID *int64 `bun:"parent_id" json:"parentId,omitempty"`
	// Kind is income or expense, the kind of the ledger accounts whose
	// postings may carry the category. It never changes.
	Kind      LedgerAccountKind `bun:"kind,notnull" json:"kind"`
	Name      string            `bun:"name,notnull" json:"name"`
	ClosedAt  *time.Time        `bun:"closed_at,nullzero" json:"closedAt,omitempty"`
	CreatedAt time.Time         `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time         `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	Parent *Category `bun:"rel:belongs-to,join:parent_id=id,on_delete:restrict" json:"parent,omitempty"`
}

// IsCategoryKind reports whether a category can have the kind: income or
// expense.
func IsCategoryKind(kind LedgerAccountKind) bool {
	return kind == LedgerAccountIncome || kind == LedgerAccountExpense
}
