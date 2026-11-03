package assetvalue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ierrors "github.com/logicorg/ivfs/internal/pkg/errors"
)

// Source is a provider under the name it is configured, shown and
// remembered by.
type Source struct {
	Name     string
	Provider Provider
}

// Sources are the configured providers, in order of preference, and how
// long each one has to answer.
type Sources struct {
	Providers []Source
	Timeout   time.Duration
}

// Answer is how a provider answered: with a series, or why it did not.
type Answer[T any] struct {
	Provider string
	Series   []T
	// Failure is why the provider did not answer, as the UI shows it, and
	// empty when it did.
	Failure string
}

// Ask asks every provider at once with ask, each bounded by the timeout, and
// returns their answers in the providers' order. A provider with nothing to
// say, which returns ErrNotFound, is left out: it is how a provider declines
// an asset or a pair.
func Ask[T any](ctx context.Context, sources Sources, ask func(context.Context, Provider) ([]T, error)) []Answer[T] {
	series := make([][]T, len(sources.Providers))
	errs := make([]error, len(sources.Providers))
	timedOut := make([]bool, len(sources.Providers))
	var wg sync.WaitGroup
	for i, source := range sources.Providers {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(ctx, sources.Timeout)
			defer cancel()
			series[i], errs[i] = ask(ctx, source.Provider)
			// The deadline is read from the context, not the error: a
			// provider may report it as anything, as the manual one's
			// workflow failing.
			timedOut[i] = errors.Is(ctx.Err(), context.DeadlineExceeded)
		})
	}
	wg.Wait()

	var answers []Answer[T]
	for i, source := range sources.Providers {
		var notFound ierrors.ErrNotFound
		answer := Answer[T]{Provider: source.Name}
		switch err := errs[i]; {
		case err == nil:
			answer.Series = series[i]
		case errors.As(err, &notFound):
			continue
		case timedOut[i] && ctx.Err() == nil:
			answer.Failure = fmt.Sprintf("no answer within %s", sources.Timeout)
		default:
			answer.Failure = err.Error()
		}
		answers = append(answers, answer)
	}
	return answers
}
