package tenancy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

// HeaderKey is the Temporal header that carries the user's id.
const HeaderKey = "ivfs-user"

// Propagator copies the user between contexts and Temporal headers: from the
// gateway's request into the start header, from the header into the
// workflow, and from the workflow into its activities, children, and
// continue-as-new. Every client and worker must register it, or the user is
// silently dropped on the way.
func Propagator() workflow.ContextPropagator {
	return propagator{}
}

type propagator struct{}

// A context without a user writes no header, rather than failing: the worker
// rejects such a run with a message naming the workflow, which an error here
// could not.
func (propagator) Inject(ctx context.Context, w workflow.HeaderWriter) error {
	return inject(ctx.Value(userKey{}), w)
}

func (propagator) InjectFromWorkflow(ctx workflow.Context, w workflow.HeaderWriter) error {
	return inject(ctx.Value(userKey{}), w)
}

func (propagator) Extract(ctx context.Context, r workflow.HeaderReader) (context.Context, error) {
	uid, ok, err := extract(r)
	if err != nil || !ok {
		return ctx, err
	}
	return context.WithValue(ctx, userKey{}, uid), nil
}

func (propagator) ExtractToWorkflow(ctx workflow.Context, r workflow.HeaderReader) (workflow.Context, error) {
	uid, ok, err := extract(r)
	if err != nil || !ok {
		return ctx, err
	}
	return workflow.WithValue(ctx, userKey{}, uid), nil
}

func inject(v any, w workflow.HeaderWriter) error {
	uid, ok := user(v)
	if !ok {
		return nil
	}
	payload, err := converter.GetDefaultDataConverter().ToPayload(uid.String())
	if err != nil {
		return fmt.Errorf("encode %s header: %w", HeaderKey, err)
	}
	w.Set(HeaderKey, payload)
	return nil
}

// extract fails on a header it cannot read, so a malformed user never
// degrades into no user, nor into the nil UUID.
func extract(r workflow.HeaderReader) (uuid.UUID, bool, error) {
	payload, ok := r.Get(HeaderKey)
	if !ok {
		return uuid.Nil, false, nil
	}
	var s string
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &s); err != nil {
		return uuid.Nil, false, fmt.Errorf("decode %s header: %w", HeaderKey, err)
	}
	uid, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("parse %s header %q: %w", HeaderKey, s, err)
	}
	if uid == uuid.Nil {
		return uuid.Nil, false, fmt.Errorf("%s header holds the nil UUID", HeaderKey)
	}
	return uid, true, nil
}
