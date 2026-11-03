package temporalnamespace

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
)

// fakeClient answers Describe and Register with the errors it is given, and
// records what was registered.
type fakeClient struct {
	describeErr, registerErr error
	registered               *workflowservice.RegisterNamespaceRequest
}

func (f *fakeClient) Describe(context.Context, string) (*workflowservice.DescribeNamespaceResponse, error) {
	if f.describeErr != nil {
		return nil, f.describeErr
	}
	return &workflowservice.DescribeNamespaceResponse{}, nil
}

func (f *fakeClient) Register(_ context.Context, request *workflowservice.RegisterNamespaceRequest) error {
	f.registered = request
	return f.registerErr
}

// TestEnsure checks that only a missing namespace is registered, that losing
// the race to register it is success, and that any other failure is reported.
func TestEnsure(t *testing.T) {
	retention := 30 * 24 * time.Hour

	tests := map[string]struct {
		describeErr, registerErr error
		wantCreated              bool
		wantRegistered           bool
		wantErr                  string
	}{
		"the namespace exists": {},
		"the namespace is missing": {
			describeErr: serviceerror.NewNamespaceNotFound("ivfs"),
			wantCreated: true, wantRegistered: true,
		},
		"another run registers it first": {
			describeErr:    serviceerror.NewNamespaceNotFound("ivfs"),
			registerErr:    serviceerror.NewNamespaceAlreadyExists("ivfs"),
			wantRegistered: true,
		},
		"Temporal is unreachable": {
			describeErr: serviceerror.NewUnavailable("connection refused"),
			wantErr:     "describe namespace ivfs: connection refused",
		},
		"the registration is refused": {
			describeErr:    serviceerror.NewNamespaceNotFound("ivfs"),
			registerErr:    serviceerror.NewPermissionDenied("not an admin", ""),
			wantRegistered: true,
			wantErr:        "register namespace ivfs: not an admin",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &fakeClient{describeErr: tt.describeErr, registerErr: tt.registerErr}

			created, err := Ensure(context.Background(), c, "ivfs", retention)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantCreated, created)
			if !tt.wantRegistered {
				require.Nil(t, c.registered)
				return
			}
			require.Equal(t, "ivfs", c.registered.GetNamespace())
			require.Equal(t, retention, c.registered.GetWorkflowExecutionRetentionPeriod().AsDuration())
		})
	}
}

