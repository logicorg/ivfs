package tenancy

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestUserFrom checks that only a non-nil user set by WithUser comes back,
// so a zero value never passes for someone.
func TestUserFrom(t *testing.T) {
	uid := uuid.Must(uuid.NewV7())

	tests := map[string]struct {
		ctx    func(context.Context) context.Context
		want   uuid.UUID
		wantOK bool
	}{
		"no user": {
			ctx: func(ctx context.Context) context.Context { return ctx },
		},
		"the nil UUID": {
			ctx: func(ctx context.Context) context.Context { return WithUser(ctx, uuid.Nil) },
		},
		"a value of another type under a lookalike key": {
			ctx: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, struct{}{}, uid) //nolint: staticcheck // the point is a key that is not userKey
			},
		},
		"a user": {
			ctx:    func(ctx context.Context) context.Context { return WithUser(ctx, uid) },
			want:   uid,
			wantOK: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := UserFrom(tc.ctx(t.Context()))
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.want, got)
		})
	}
}
