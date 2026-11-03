// Package temporalnamespace creates the Temporal namespace ivfs runs in, for
// a cluster where nobody else does (the Kubernetes deployment's Job).
package temporalnamespace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Client is the part of client.NamespaceClient that Ensure needs.
type Client interface {
	Describe(ctx context.Context, name string) (*workflowservice.DescribeNamespaceResponse, error)
	Register(ctx context.Context, request *workflowservice.RegisterNamespaceRequest) error
}

// Ensure registers the namespace name with retention unless it exists, and
// reports whether it did. An existing namespace is left as it is, retention
// included: changing one is an operator's decision, not a deployment's.
func Ensure(ctx context.Context, c Client, name string, retention time.Duration) (bool, error) {
	_, err := c.Describe(ctx, name)
	if err == nil {
		return false, nil
	}
	if _, ok := errors.AsType[*serviceerror.NamespaceNotFound](err); !ok {
		return false, fmt.Errorf("describe namespace %s: %w", name, err)
	}

	err = c.Register(ctx, &workflowservice.RegisterNamespaceRequest{
		Namespace:                        name,
		WorkflowExecutionRetentionPeriod: durationpb.New(retention),
	})
	// Another run of the Job got there between the two calls.
	if _, ok := errors.AsType[*serviceerror.NamespaceAlreadyExists](err); ok {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("register namespace %s: %w", name, err)
	}
	return true, nil
}
