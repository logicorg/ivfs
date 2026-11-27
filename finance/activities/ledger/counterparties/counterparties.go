// Package counterparties holds the ledger activities that manage counterparties.
package counterparties

import "github.com/logicorg/ivfs/internal/pkg/tenancy"

// Activities groups the counterparty activities so Temporal registers them together.
type Activities struct{ Pools tenancy.Pools }
