package currency

import (
	"strings"

	"github.com/logicorg/ivfs/internal/pkg/tenancy"
)

// Currency holds the currency activities
type Currency struct {
	Pools tenancy.Pools
}

// normaliseCode returns a currency code in its one canonical form, trimmed and
// upper-cased (ADR 0005). Every write and every lookup by code goes through
// it, so the stored code and the code looked up always compare exactly.
func normaliseCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
