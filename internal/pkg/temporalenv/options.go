// Package temporalenv reads where Temporal is from the environment, the same
// way for the worker, the gateway and the MCP server, so the three can never
// disagree on which namespace they use.
package temporalenv

import (
	"fmt"
	"net"

	"go.temporal.io/sdk/client"
)

// The variables Options reads.
const (
	HostPortVariable  = "TEMPORAL_HOST_PORT"
	NamespaceVariable = "TEMPORAL_NAMESPACE"
)

// Options returns the client options the environment sets, through getenv.
// Unset, Temporal is on localhost and the namespace is default, which is what
// the development compose runs.
func Options(getenv func(string) string) (client.Options, error) {
	hostPort := getenv(HostPortVariable)
	if hostPort == "" {
		hostPort = client.DefaultHostPort
	}
	if _, _, err := net.SplitHostPort(hostPort); err != nil {
		return client.Options{}, fmt.Errorf("invalid %s %q, expected host:port such as temporal-frontend:7233: %w",
			HostPortVariable, hostPort, err)
	}

	namespace := getenv(NamespaceVariable)
	if namespace == "" {
		namespace = client.DefaultNamespace
	}
	return client.Options{HostPort: hostPort, Namespace: namespace}, nil
}
