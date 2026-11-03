package assetvalue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

// stub answers with values or err, or waits for its context to end when
// stalled, as a provider that never answers.
type stub struct {
	values  []Value
	err     error
	stalled bool
}

func (s stub) Values(ctx context.Context, _ model.Asset, _ Range) ([]Value, error) {
	if s.stalled {
		<-ctx.Done()
		// Reported as something else, as the manual provider reports its
		// workflow's deadline.
		return nil, fmt.Errorf("the workflow failed")
	}
	return s.values, s.err
}

func (stub) Rates(context.Context, string, string, Range) ([]Rate, error) {
	return nil, errors.ErrNotFound{ResourceType: "Conversion"}
}

func TestAsk(t *testing.T) {
	value := Value{Date: date(2026, 10, 9), Currency: "USD"}
	values := func(ctx context.Context, p Provider) ([]Value, error) {
		return p.Values(ctx, model.Asset{}, Range{})
	}

	tests := map[string]struct {
		providers   []Source
		wantAnswers []Answer[Value]
	}{
		"no providers": {},
		"every one answers, in the configured order": {
			providers: []Source{
				{Name: "manual", Provider: stub{values: []Value{value}}},
				{Name: "yahoo", Provider: stub{values: []Value{value, value}}},
			},
			wantAnswers: []Answer[Value]{
				{Provider: "manual", Series: []Value{value}},
				{Provider: "yahoo", Series: []Value{value, value}},
			},
		},
		"one has no value, so it is left out": {
			providers: []Source{
				{Name: "manual", Provider: stub{err: errors.ErrNotFound{ResourceType: "AssetValue"}}},
				{Name: "yahoo", Provider: stub{values: []Value{value}}},
			},
			wantAnswers: []Answer[Value]{{Provider: "yahoo", Series: []Value{value}}},
		},
		"one fails, and the others still answer": {
			providers: []Source{
				{Name: "broken", Provider: stub{err: fmt.Errorf("call the provider: connection refused")}},
				{Name: "yahoo", Provider: stub{values: []Value{value}}},
			},
			wantAnswers: []Answer[Value]{
				{Provider: "broken", Failure: "call the provider: connection refused"},
				{Provider: "yahoo", Series: []Value{value}},
			},
		},
		"one never answers, and is cut off at the timeout": {
			providers: []Source{
				{Name: "slow", Provider: stub{stalled: true}},
				{Name: "yahoo", Provider: stub{values: []Value{value}}},
			},
			wantAnswers: []Answer[Value]{
				{Provider: "slow", Failure: "no answer within 50ms"},
				{Provider: "yahoo", Series: []Value{value}},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			answers := Ask(t.Context(), Sources{Providers: tc.providers, Timeout: 50 * time.Millisecond}, values)
			require.Equal(t, tc.wantAnswers, answers)
		})
	}
}
