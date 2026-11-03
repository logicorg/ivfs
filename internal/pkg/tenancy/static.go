package tenancy

import (
	"context"

	"github.com/uptrace/bun"
)

// Static returns Pools that hand out db for any context, with or without a
// user. Only tests may use it: it ignores who is asking (ADR 0013).
func Static(db *bun.DB) Pools {
	return static{db: db}
}

type static struct {
	db *bun.DB
}

func (s static) For(context.Context) (*bun.DB, error) {
	return s.db, nil
}
