package reporting

import "github.com/logicorg/ivfs/internal/pkg/tenancy"

// Activities is a struct holding ledger reporting (Temporal) activities.
type Activities struct{ Pools tenancy.Pools }
