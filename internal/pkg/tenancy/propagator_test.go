package tenancy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// header is a HeaderReader and HeaderWriter over a plain map.
type header map[string]*commonpb.Payload

func (h header) Set(k string, v *commonpb.Payload) { h[k] = v }

func (h header) Get(k string) (*commonpb.Payload, bool) {
	v, ok := h[k]
	return v, ok
}

func (h header) ForEachKey(f func(string, *commonpb.Payload) error) error {
	for k, v := range h {
		if err := f(k, v); err != nil {
			return err
		}
	}
	return nil
}

func payload(t *testing.T, v any) *commonpb.Payload {
	t.Helper()
	p, err := converter.GetDefaultDataConverter().ToPayload(v)
	require.NoError(t, err)
	return p
}

// TestPropagatorRoundTrip checks that the user crosses a header unchanged,
// and that no user writes no header rather than an empty one.
func TestPropagatorRoundTrip(t *testing.T) {
	uid := uuid.Must(uuid.NewV7())

	tests := map[string]struct {
		ctx        context.Context
		wantHeader bool
		wantOK     bool
	}{
		"no user":      {ctx: t.Context()},
		"the nil UUID": {ctx: WithUser(t.Context(), uuid.Nil)},
		"a user":       {ctx: WithUser(t.Context(), uid), wantHeader: true, wantOK: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			h := header{}
			require.NoError(t, Propagator().Inject(tc.ctx, h))
			_, ok := h[HeaderKey]
			require.Equal(t, tc.wantHeader, ok)

			ctx, err := Propagator().Extract(t.Context(), h)
			require.NoError(t, err)
			got, ok := UserFrom(ctx)
			require.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				require.Equal(t, uid, got)
			}
		})
	}
}

// TestPropagatorExtractRejects checks that a header that is there but is not
// a user fails, instead of passing as no user.
func TestPropagatorExtractRejects(t *testing.T) {
	tests := map[string]struct {
		payload *commonpb.Payload
		wantErr string
	}{
		"not a string":  {payload: payload(t, 42), wantErr: "decode ivfs-user header"},
		"not a UUID":    {payload: payload(t, "alice"), wantErr: `parse ivfs-user header "alice"`},
		"the nil UUID":  {payload: payload(t, uuid.Nil.String()), wantErr: "ivfs-user header holds the nil UUID"},
		"empty payload": {payload: &commonpb.Payload{}, wantErr: "decode ivfs-user header"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Propagator().Extract(t.Context(), header{HeaderKey: tc.payload})
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// probe is what each step of probeWorkflow saw as its user, "" for none.
type probe struct {
	Workflow, Activity, Child string
}

func seen(uid uuid.UUID, ok bool) string {
	if !ok {
		return ""
	}
	return uid.String()
}

func probeActivity(ctx context.Context) (string, error) {
	return seen(UserFrom(ctx)), nil
}

func probeChildWorkflow(ctx workflow.Context) (string, error) {
	return seen(WorkflowUser(ctx)), nil
}

func probeWorkflow(ctx workflow.Context) (probe, error) {
	var p probe
	p.Workflow = seen(WorkflowUser(ctx))

	actx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Second})
	if err := workflow.ExecuteActivity(actx, probeActivity).Get(ctx, &p.Activity); err != nil {
		return probe{}, err
	}
	if err := workflow.ExecuteChildWorkflow(ctx, probeChildWorkflow).Get(ctx, &p.Child); err != nil {
		return probe{}, err
	}
	return p, nil
}

func continueWorkflow(ctx workflow.Context) error {
	return workflow.NewContinueAsNewError(ctx, probeWorkflow)
}

// TestPropagatorInWorkflows checks that the user in a start header reaches
// the workflow, its activities, its children, and its next run after
// continue-as-new, which the SDK carries only through registered propagators.
func TestPropagatorInWorkflows(t *testing.T) {
	uid := uuid.Must(uuid.NewV7())

	tests := map[string]struct {
		header header
		want   string
	}{
		"no user": {header: header{}},
		"a user":  {header: header{HeaderKey: payload(t, uid.String())}, want: uid.String()},
	}

	for name, tc := range tests {
		newEnv := func() *testsuite.TestWorkflowEnvironment {
			var s testsuite.WorkflowTestSuite
			s.SetContextPropagators([]workflow.ContextPropagator{Propagator()})
			env := s.NewTestWorkflowEnvironment()
			env.SetHeader(&commonpb.Header{Fields: tc.header})
			return env
		}

		t.Run(name+"/workflow, activity, child", func(t *testing.T) {
			env := newEnv()
			env.RegisterActivity(probeActivity)
			env.RegisterWorkflow(probeChildWorkflow)

			env.ExecuteWorkflow(probeWorkflow)
			require.NoError(t, env.GetWorkflowError())
			var got probe
			require.NoError(t, env.GetWorkflowResult(&got))
			require.Equal(t, probe{Workflow: tc.want, Activity: tc.want, Child: tc.want}, got)
		})

		t.Run(name+"/continue-as-new", func(t *testing.T) {
			env := newEnv()
			env.ExecuteWorkflow(continueWorkflow)

			var can *workflow.ContinueAsNewError
			require.True(t, errors.As(env.GetWorkflowError(), &can))
			ctx, err := Propagator().Extract(t.Context(), header(can.Header.GetFields()))
			require.NoError(t, err)
			require.Equal(t, tc.want, seen(UserFrom(ctx)))
		})
	}
}
