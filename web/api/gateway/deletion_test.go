package gateway_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/logicorg/ivfs/web/api/gateway"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

// TestRunDeletion checks that a deletion is started under its ID, joining a
// running one, without an execution timeout, and that a deletion still running
// when the wait ends is reported as running rather than failed.
func TestRunDeletion(t *testing.T) {
	tests := map[string]struct {
		startErr error
		runErr   error
		// blocks makes the run outlast the wait.
		blocks      bool
		wantRunning bool
		wantStatus  int
	}{
		"a deletion that finishes": {},
		"a deletion still running": {
			blocks:      true,
			wantRunning: true,
		},
		"a deletion that fails": {
			runErr:     temporal.NewApplicationError("resource LedgerAccount conflict: still referenced", "ErrResourceConflict"),
			wantStatus: http.StatusConflict,
		},
		"a deletion that could not start": {
			startErr:   errors.New("temporal unavailable"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			run := &mocks.WorkflowRun{}
			run.On("Get", mock.Anything, mock.Anything).Return(func(ctx context.Context, result any) error {
				if tc.blocks {
					<-ctx.Done()
					return ctx.Err()
				}
				*result.(*string) = "out"
				return tc.runErr
			}).Maybe()
			c := &mocks.Client{}
			c.On("ExecuteWorkflow", mock.Anything, client.StartWorkflowOptions{
				ID: tenancy.WorkflowIDPrefix(apitest.User) + "delete-ledger-account-7", TaskQueue: workflowrun.TaskQueueName,
				WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
			}, mock.Anything, "in").Return(run, tc.startErr)

			result, running, err := gateway.RunDeletionWithin[string](tenancy.WithUser(t.Context(), apitest.User), c, 10*time.Millisecond, "delete-ledger-account-7", echoWorkflow, "in")

			c.AssertExpectations(t)
			if tc.wantStatus != 0 {
				var problem *huma.ErrorModel
				require.ErrorAs(t, err, &problem)
				require.Equal(t, tc.wantStatus, problem.Status)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantRunning, running)
			if tc.wantRunning {
				require.Nil(t, result)
				return
			}
			require.Equal(t, "out", *result)
		})
	}
}

// TestNewAccountDeletionOutput checks that a finished deletion is answered with
// its warnings, and one still running with 202 Accepted and its workflow ID
// as the request named it, without the user's prefix.
func TestNewAccountDeletionOutput(t *testing.T) {
	warnings := []lt.NegativeHolding{{FinancialAccountID: 3, AssetID: 12, Units: decimal.NewFromInt(-10)}}
	tests := map[string]struct {
		running bool
		want    *gateway.AccountDeletionOutput
	}{
		"a deletion that finished": {
			want: &gateway.AccountDeletionOutput{Status: http.StatusOK, Body: gateway.AccountDeletion{Warnings: gateway.NewNegativeHoldings(warnings)}},
		},
		"a deletion still running": {
			running: true,
			want:    &gateway.AccountDeletionOutput{Status: http.StatusAccepted, Body: gateway.AccountDeletion{WorkflowID: "delete-ledger-account-7"}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, gateway.NewAccountDeletionOutput("delete-ledger-account-7", tc.running, warnings))
		})
	}
}

// TestReplacement checks that a request without a replacement passes none.
func TestReplacement(t *testing.T) {
	require.Nil(t, gateway.Replacement(0))
	require.Equal(t, ptr.Int64(8), gateway.Replacement(8))
}
