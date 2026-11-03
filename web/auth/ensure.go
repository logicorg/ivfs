package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/provisioning/common/types/usertypes"
	"github.com/logicorg/ivfs/provisioning/workflows"
	"github.com/logicorg/ivfs/provisioning/workflows/users"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// ensureTimeout bounds a sign-in's wait for its account; a first sign-in
// creates a database, which takes a few seconds, not minutes.
const ensureTimeout = time.Minute

// EnsureUserWith returns the EnsureUser that runs the provisioning workflow
// on c.
//
// It starts the workflow directly rather than through workflowrun: the
// signing-in user has no session yet, and the finance queue is the wrong one.
func EnsureUserWith(c client.Client) EnsureUser {
	return func(ctx context.Context, issuer, subject string) (uuid.UUID, error) {
		// One ID per identity, so two tabs signing in at once join one run
		// rather than create the same database side by side. The ID is a
		// hash, so the subject, which may be an email, is not in it.
		identity := sha256.Sum256([]byte(issuer + "\x00" + subject))
		options := client.StartWorkflowOptions{
			ID:                       "ensure-user/" + hex.EncodeToString(identity[:]),
			TaskQueue:                workflows.TaskQueueName,
			WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
			WorkflowExecutionTimeout: ensureTimeout,
		}
		run, err := c.ExecuteWorkflow(ctx, options, users.EnsureUser, usertypes.EnsureUserParams{Issuer: issuer, Subject: subject})
		if err != nil {
			return uuid.Nil, err
		}
		var result usertypes.EnsureUserResult
		if err := run.Get(ctx, &result); err != nil {
			return uuid.Nil, err
		}
		return result.ID, nil
	}
}
