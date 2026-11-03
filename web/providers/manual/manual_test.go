package manual

import (
	"context"
	"testing"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/mocks"
)

// provider returns a manual provider whose client expects calls, each once
// and nothing else, and a context acting as apitest.User.
func provider(t *testing.T, calls ...*apitest.WorkflowCall) (*Provider, context.Context) {
	t.Helper()
	c := &mocks.Client{}
	for _, call := range calls {
		call.Expect(t, c)
	}
	t.Cleanup(func() { c.AssertExpectations(t) })
	return New(c), tenancy.WithUser(context.Background(), apitest.User)
}

// listCurrencies is the user's currencies being listed.
func listCurrencies() *apitest.WorkflowCall {
	return &apitest.WorkflowCall{
		Workflow: currencieswf.ListCurrencies, Params: ct.ListCurrenciesParams{},
		Result: ct.ListCurrenciesResult{Currencies: []*model.Currency{
			{ID: 1, Code: "EUR", Name: "Euro"},
			{ID: 2, Code: "USD", Name: "US Dollar"},
		}},
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
