// Package settings holds the activities over a user's settings (ADR 0015).
package settings

import "github.com/logicorg/ivfs/internal/pkg/tenancy"

// Activities reads and writes the settings row of the user's database.
type Activities struct {
	Pools tenancy.Pools
}
