// Package deletion is the activities that empty and remove an account: closing
// it, moving its history to the stray accounts, and previewing the outcome.
package deletion

import "github.com/logicorg/ivfs/internal/pkg/tenancy"

// Activities holds the account deletion activities.
type Activities struct{ Pools tenancy.Pools }
