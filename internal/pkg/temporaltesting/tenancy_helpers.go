package temporaltesting

import (
	"testing"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// RunAs makes env run its workflow as uid, as if the gateway had started it:
// the user is in the start header, and the propagator carries it into the
// workflow and on to its children and activities.
func RunAs(t testing.TB, env *testsuite.TestWorkflowEnvironment, uid uuid.UUID) {
	t.Helper()
	payload, err := converter.GetDefaultDataConverter().ToPayload(uid.String())
	require.NoError(t, err)
	env.SetContextPropagators([]workflow.ContextPropagator{tenancy.Propagator()})
	env.SetHeader(&commonpb.Header{Fields: map[string]*commonpb.Payload{tenancy.HeaderKey: payload}})
}
