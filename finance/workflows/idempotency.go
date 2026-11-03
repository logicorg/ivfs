package workflows

import (
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// DefaultIdempotencyKey returns key unchanged when the caller supplied one,
// and otherwise generates a random one, so that a caller with no natural way
// to invent a stable key (an LLM composing a command, for example) still gets
// one write per execution. Activities keep requiring and validating a
// non-empty key; this only fills it in before the workflow calls one.
//
// The key is deliberately not derived from the workflow's own execution ID:
// that would force whoever starts these workflows to choose IDs with the
// right idempotency semantics in mind, coupling two unrelated concerns.
// Instead it is generated inside workflow.SideEffect, which runs the given
// function once - on whichever worker happens to execute this point in the
// workflow for the first time, using that worker's own randomness - and
// records the result in workflow history as a Marker event. Every later
// replay of this execution returns that recorded value instead of calling
// uuid.NewString() again, which is what keeps this deterministic: calling
// uuid.NewString() directly from workflow code, without SideEffect, is not
// safe, since replaying the same history would then produce a different key
// than the original run did.
func DefaultIdempotencyKey(ctx workflow.Context, key string) string {
	if key != "" {
		return key
	}

	var generated string
	// Decoding a plain string that we encoded ourselves via the workflow's own
	// data converter cannot fail, so the error is not worth threading through
	// every call site.
	_ = workflow.SideEffect(ctx, func(workflow.Context) interface{} {
		return uuid.NewString()
	}).Get(&generated)
	return generated
}
