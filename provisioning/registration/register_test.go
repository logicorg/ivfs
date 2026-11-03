package registration

import (
	"testing"

	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestRegister checks that nothing is registered twice: the worker panics at
// start on a duplicate activity or workflow, and so does this test.
func TestRegister(t *testing.T) {
	env := new(testsuite.WorkflowTestSuite).NewTestWorkflowEnvironment()

	require.NotPanics(t, func() { Register(env, nil, registry.Server{}) })
}
