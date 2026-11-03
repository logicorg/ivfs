package tenancy

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestStatic(t *testing.T) {
	db := &bun.DB{}

	tests := map[string]struct {
		ctx context.Context
	}{
		"no user": {ctx: t.Context()},
		"a user":  {ctx: WithUser(t.Context(), uuid.Must(uuid.NewV7()))},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := Static(db).For(tc.ctx)
			require.NoError(t, err)
			require.Same(t, db, got)
		})
	}
}
