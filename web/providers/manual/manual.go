// Package manual is the built-in asset value provider (ADR 0014): the values
// and rates the user entered by hand, read from their database through the
// finance workflows rather than over the provider spec.
package manual

import (
	"context"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

// Name is the name the manual provider is configured and shown under.
const Name = "manual"

// Provider reads the values and rates of the user each call's context acts
// as.
type Provider struct {
	c client.Client
}

var _ assetvalue.Provider = (*Provider)(nil)

// New returns the manual provider, running its workflows through c.
func New(c client.Client) *Provider {
	return &Provider{c: c}
}

// currencyCodes maps the user's currency IDs to their codes, which the
// provider interface speaks.
func (p *Provider) currencyCodes(ctx context.Context) (map[int64]string, error) {
	result, err := gateway.RunWorkflow[ct.ListCurrenciesResult](ctx, p.c, currencieswf.ListCurrencies, ct.ListCurrenciesParams{})
	if err != nil {
		return nil, err
	}
	codes := make(map[int64]string, len(result.Currencies))
	for _, currency := range result.Currencies {
		codes[currency.ID] = currency.Code
	}
	return codes, nil
}
