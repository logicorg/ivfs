package gateway

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"go.temporal.io/sdk/client"
)

// RunDeletion runs the deletion wf with in under workflowID, behind the
// user's prefix, joining the deletion already running under it, and waits at most WorkflowTimeout for
// its result (ADR 0012). A deletion still running by then is not cut short:
// running is true, and asking again joins it.
func RunDeletion[O any](ctx context.Context, c client.Client, workflowID string, wf, in any) (result *O, running bool, err error) {
	return runDeletion[O](ctx, c, WorkflowTimeout, workflowID, wf, in)
}

func runDeletion[O any](ctx context.Context, c client.Client, wait time.Duration, workflowID string, wf, in any) (*O, bool, error) {
	run, err := workflowrun.Start(ctx, c, wf, in, workflowrun.Joining(workflowID))
	if err != nil {
		return nil, false, WorkflowError(err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	result := new(O)
	if err := run.Get(waitCtx, result); err != nil {
		// Only the wait ran out, not the request, so the deletion is still
		// running rather than failed.
		if ctx.Err() == nil && errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return nil, true, nil
		}
		return nil, false, WorkflowError(err)
	}
	return result, false, nil
}

// DeleteAccount runs the deletion wf with in under workflowID, as RunDeletion
// does, and answers with the warnings its result holds, or with 202 Accepted
// while it is still running.
func DeleteAccount[O any](ctx context.Context, c client.Client, workflowID string, wf, in any, warnings func(*O) []lt.NegativeHolding) (*AccountDeletionOutput, error) {
	result, running, err := RunDeletion[O](ctx, c, workflowID, wf, in)
	if err != nil {
		return nil, err
	}
	return NewAccountDeletionOutput(workflowID, running, warnings(result)), nil
}

// NewAccountDeletionOutput is the answer to a deletion under workflowID that
// is still running, or that finished with warnings. workflowID is the one
// the endpoint named: the user's prefix is the worker's business, not the
// API's, and stays out of the answer.
func NewAccountDeletionOutput(workflowID string, running bool, warnings []lt.NegativeHolding) *AccountDeletionOutput {
	if running {
		return &AccountDeletionOutput{Status: http.StatusAccepted, Body: AccountDeletion{WorkflowID: workflowID}}
	}
	return &AccountDeletionOutput{Status: http.StatusOK, Body: AccountDeletion{Warnings: NewNegativeHoldings(warnings)}}
}

// AccountDeletionResponses documents the 202 Accepted a deletion answers while
// it is still running, as an operation's Responses.
func AccountDeletionResponses(api huma.API) map[string]*huma.Response {
	schema := api.OpenAPI().Components.Schemas.Schema(reflect.TypeFor[AccountDeletion](), true, "")
	return map[string]*huma.Response{
		strconv.Itoa(http.StatusAccepted): {
			Description: "The deletion is still running; deleting again waits on it",
			Content:     map[string]*huma.MediaType{"application/json": {Schema: schema}},
		},
	}
}

// Replacement is a replacementId query parameter as the deletion workflows
// take it: nil when the request has none.
func Replacement(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}
