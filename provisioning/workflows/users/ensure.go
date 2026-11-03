// Package users contains the workflows that provision users (ADR 0013).
package users

import (
	"github.com/logicorg/ivfs/provisioning/activities/user"
	"github.com/logicorg/ivfs/provisioning/common/types/usertypes"
	"github.com/logicorg/ivfs/provisioning/workflows"
	"go.temporal.io/sdk/workflow"
)

// EnsureUser returns the user an identity belongs to, registering it and
// making its database ready on its first login. Every step is idempotent, so
// a login that fails halfway is finished by the next one, and a provisioned
// user skips straight to the answer.
func EnsureUser(ctx workflow.Context, params usertypes.EnsureUserParams) (*usertypes.EnsureUserResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.DefaultActivityOptions)

	var a *user.Activities
	var inserted usertypes.InsertUserResult
	if err := workflow.ExecuteActivity(ctx, a.InsertUser, usertypes.InsertUserParams(params)).Get(ctx, &inserted); err != nil {
		return nil, err
	}

	if !inserted.Provisioned {
		if err := workflow.ExecuteActivity(ctx, a.CreateUserDatabase, inserted.ID).Get(ctx, nil); err != nil {
			return nil, err
		}
		if err := workflow.ExecuteActivity(ctx, a.ProvisionUser, inserted.ID).Get(ctx, nil); err != nil {
			return nil, err
		}
	}

	return &usertypes.EnsureUserResult{ID: inserted.ID}, nil
}
