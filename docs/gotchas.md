# Gotchas

## Temporal Error Casting

When a Temporal activity returns an error, the caller code cannot directly convert the received error back to its original Go type. This is a deliberate design choice by Temporal. Since callers and callees can be implemented in different programming languages, direct type casting would be problematic or impossible.

Instead, Temporal provides a mechanism to inspect the error type if it's an `ApplicationError`. The `ApplicationError` has a `.Type()` method that returns a string representation of the original error's type name.

For example, if an activity returns `errors.ErrNotFound`, the caller can check for this by inspecting the string returned by the `.Type()` method.

### Example

Here is how you can test for a specific application error returned from a Temporal activity:

```go
import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestMyWorkflow(t *testing.T) {
	// ... test setup ...

	// Assuming 'err' is the error returned from a Temporal activity
	var err error 

	// Example test case where we expect a specific error type
	expectedErrType := "ErrNotFound" 

	if expectedErrType != "" {
		appErr := new(temporal.ApplicationError)
		// Check if the error is a temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		// Check if the type of the application error matches our expectation
		assert.Equal(t, expectedErrType, appErr.Type())
	}
}
```

This allows for error matching in a language-agnostic way, which is a core design principle of Temporal.

## Register cuego constraints once

`cuego.MustConstrain` stores the constraint in global state and unifies it with
every constraint already registered for that type. Calling it inside a
`Validate` method therefore makes each call slower than the last, by roughly a
factor of five: after about a dozen calls a single validation took seconds, and
after fourteen it took 46 seconds. A long-running worker would slow down until
it was restarted.

Register the constraint once, in an `init` function, and keep `Validate` to
`cuego.Complete` and `cuego.Validate`. See `RetrieveInstitutionParams` in
`finance/common/types/institutiontypes/retrieve.go`.

## Return structured errors unwrapped

Temporal reports the type of an activity error from the Go type of the error the
activity returns. `errors.ErrNotFound` reports `ErrNotFound`, which the
workflows list as non-retryable. The same error wrapped with
`fmt.Errorf("...: %w", err)` reports `wrapError`, which matches nothing, so
Temporal retries it as if it were transient: five attempts and about fifteen
seconds of backoff for an error that can never succeed.

Wrap only genuine infrastructure errors, and return a structured error as it
is. That is why `sqllib.ParentNotFound` and `sqllib.StillReferenced` return the
structured error, or nil when the error is something else, so the caller can
fall back to wrapping:

```go
if notFound := sqllib.ParentNotFound(err); notFound != nil {
	return notFound
}
return fmt.Errorf("insert financial account: %w", err)
```

## Error types through a child workflow

A facade runs lower-level workflows as children (ADR 0006), and those
workflows wrap their activity's error with `fmt.Errorf("...: %w", err)`.
Temporal keeps the whole cause chain, so the structured error still reaches
the facade, but not first: the chain is the child's
`ChildWorkflowExecutionError`, then an `ApplicationError` of type `wrapError`
for the child's wrapper, then the `ActivityError`, and only then the
`ApplicationError` of type `ErrNotFound`.

A single `errors.As` stops at the first `ApplicationError`, the `wrapError`
one, and so never sees `ErrNotFound`. Use `workflows.HasErrorType`, which
walks every `ApplicationError` in the chain:

```go
err := workflow.ExecuteChildWorkflow(ctx, institutionwf.RetrieveInstitution, params).Get(ctx, &retrieved)
if workflows.HasErrorType(err, "ErrNotFound") {
	// the institution does not exist
}
```

## CUE constraints on decimal fields compare strings

`decimal.Decimal` is encoded as a string, so a numeric bound on it never
matches. `cue:"!= 0"` on a `UnitsDelta decimal.Decimal` rejected every input,
including valid ones, with:

```text
unitsDelta: conflicting values "3" and !=0 (mismatched types string and number)
```

The `2 errors in empty disjunction: (and 2 more errors)` that `Validate` prints
hides this. The full text is `cueerrors.Details(err, nil)` from
`cuelang.org/go/cue/errors`; the `null` conflict in it is only cuego reading a
pointer receiver as `struct | null`, and can be ignored.

Constrain the string instead. Zero is always encoded as `"0"`, however it was
written, so `cue:"!=\"0\""` rejects it. A constraint on a nested struct goes on
the field that holds it, for example
`cue:"{idempotencyKey: !=\"\"}"`. Rules that CUE cannot express, such as two
fields that must be given together, belong in the `Validate` method next to the
`cuego.Validate` call. See `RegisterInvestmentTradeInput`.
