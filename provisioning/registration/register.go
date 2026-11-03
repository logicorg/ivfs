// Package registration registers every activity and workflow of the
// provisioning domain, so the worker and its tests register the same set.
package registration

import (
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/logicorg/ivfs/provisioning/activities/user"
	"github.com/logicorg/ivfs/provisioning/workflows/users"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/worker"
)

// Register registers every activity, bound to the registry database and the
// server the users' databases are on, and every workflow on r, which is a
// worker or a test workflow environment.
func Register(r worker.Registry, registryDB *bun.DB, server registry.Server) {
	r.RegisterActivity(&user.Activities{Registry: registryDB, Server: server})

	r.RegisterWorkflow(users.EnsureUser)
}
