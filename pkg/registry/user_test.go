package registry_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/stretchr/testify/require"
)

// TestDatabaseName checks that the name is the uid in hex behind a fixed
// prefix, and that distinct users get distinct names.
func TestDatabaseName(t *testing.T) {
	tests := map[string]struct {
		uid  uuid.UUID
		want string
	}{
		"a uid":       {uid: uuid.MustParse("01926f3a-7b2c-7d4e-8f90-a1b2c3d4e5f6"), want: "ivfs_u_01926f3a7b2c7d4e8f90a1b2c3d4e5f6"},
		"another uid": {uid: uuid.MustParse("01926f3a-7b2c-7d4e-8f90-a1b2c3d4e5f7"), want: "ivfs_u_01926f3a7b2c7d4e8f90a1b2c3d4e5f7"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, registry.DatabaseName(tc.uid))
		})
	}
}
