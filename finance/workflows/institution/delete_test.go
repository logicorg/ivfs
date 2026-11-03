package institution

import (
	"github.com/google/uuid"
	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

func (s *WorkflowUnitTestSuite) TestDeleteInstitution() {
	user := uuid.MustParse("0192f3a0-0000-7000-8000-000000000001")
	conflict := internalerrors.ErrResourceConflict{ResourceType: "Institution", Reason: "still referenced by ledger.financial_account.institution_id"}

	tests := []struct {
		name   string
		params it.DeleteInstitutionParams
		// institution is what the retrieve returns; nil means it does not exist.
		institution *model.Institution
		retrieve    bool
		// passes holds the financial account ids each listing returns.
		passes [][]int64
		// childErr fails the deletion of the first account.
		childErr error
		// deleteErrs is what the institution's delete returns after each pass.
		deleteErrs []error
		wantErr    string
	}{
		{
			name:    "params that select nothing",
			params:  it.DeleteInstitutionParams{},
			wantErr: "validation error",
		},
		{
			name:     "an institution that does not exist",
			params:   it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{Name: "Revolut"}},
			retrieve: true,
		},
		{
			name:        "an institution without accounts",
			params:      it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{Name: "Revolut"}},
			institution: &model.Institution{ID: 7, Name: "Revolut"},
			retrieve:    true,
			passes:      [][]int64{{}},
			deleteErrs:  []error{nil},
		},
		{
			name:        "an institution with accounts",
			params:      it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: 7}},
			institution: &model.Institution{ID: 7, Name: "Revolut"},
			retrieve:    true,
			passes:      [][]int64{{11, 12, 13}},
			deleteErrs:  []error{nil},
		},
		{
			name:        "more accounts than one listing returns",
			params:      it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: 7}},
			institution: &model.Institution{ID: 7, Name: "Revolut"},
			retrieve:    true,
			passes:      [][]int64{{11, 12}, {13}},
			deleteErrs:  []error{conflict, nil},
		},
		{
			name:        "a conflict after a listing that returned nothing",
			params:      it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: 7}},
			institution: &model.Institution{ID: 7, Name: "Revolut"},
			retrieve:    true,
			passes:      [][]int64{{11}, {}},
			deleteErrs:  []error{conflict, conflict},
			wantErr:     "resource Institution conflict: still referenced by ledger.financial_account.institution_id",
		},
		{
			name:        "the untracked institution with accounts",
			params:      it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: 7}},
			institution: &model.Institution{ID: 7, Name: lt.UntrackedInstitutionName},
			retrieve:    true,
			passes:      [][]int64{{11}},
			wantErr:     "is managed by the ledger and cannot be deleted while it holds financial accounts",
		},
		{
			name:        "the untracked institution without accounts",
			params:      it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: 7}},
			institution: &model.Institution{ID: 7, Name: lt.UntrackedInstitutionName},
			retrieve:    true,
			passes:      [][]int64{{}},
			deleteErrs:  []error{nil},
		},
		{
			name:        "an account deletion that fails",
			params:      it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: 7}},
			institution: &model.Institution{ID: 7, Name: "Revolut"},
			retrieve:    true,
			passes:      [][]int64{{11, 12}},
			childErr:    internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account 9 has child accounts outside it; delete or move them first"},
			wantErr:     "resource LedgerAccount conflict: ledger account 9 has child accounts outside it",
		},
		{
			name:        "a delete that fails",
			params:      it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: 7}},
			institution: &model.Institution{ID: 7, Name: "Revolut"},
			retrieve:    true,
			passes:      [][]int64{{}},
			deleteErrs:  []error{assert.AnError},
			wantErr:     "delete institution",
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			t := s.T()
			env := s.NewTestWorkflowEnvironment()
			temporaltesting.RunAs(t, env, user)
			env.RegisterWorkflow(ledgeraccountswf.DeleteFinancialAccount)
			ia := institutionactivity.Institution{}
			la := ledgeractivity.Activities{}

			if tc.retrieve {
				retrieved := env.OnActivity(ia.Retrieve, mock.Anything, tc.params.FindBy)
				if tc.institution == nil {
					retrieved.Return(nil, internalerrors.ErrNotFound{ResourceType: "Institution", ID: tc.params.FindBy.Name}).Once()
				} else {
					retrieved.Return(&it.RetrieveInstitutionResult{Institution: tc.institution}, nil).Once()
				}
			}
			var started []string
			env.SetOnChildWorkflowStartedListener(func(info *workflow.Info, _ workflow.Context, _ converter.EncodedValues) {
				started = append(started, info.WorkflowExecution.ID)
			})
			var want []string
			for n, ids := range tc.passes {
				accounts := make([]model.FinancialAccount, len(ids))
				for i, id := range ids {
					accounts[i] = model.FinancialAccount{ID: id, InstitutionID: tc.institution.ID}
				}
				env.OnActivity(la.ListFinancialAccounts, mock.Anything, lt.ListFinancialAccountsParams{InstitutionID: &tc.institution.ID, IncludeClosed: true}).
					Return(&lt.ListFinancialAccountsResult{FinancialAccounts: accounts}, nil).Once()
				if len(ids) > 0 && lt.IsUntrackedInstitution(tc.institution.Name) {
					break
				}
				failed := false
				for _, id := range ids {
					want = append(want, tenancy.WorkflowIDPrefix(user)+workflows.DeleteFinancialAccountWorkflowID(id))
					env.OnWorkflow(ledgeraccountswf.DeleteFinancialAccount, mock.Anything, lt.DeleteFinancialAccountInput{FinancialAccountID: id}).
						Return(&lt.DeleteAccountResult{}, tc.childErr).Once()
					if tc.childErr != nil {
						failed = true
						break
					}
				}
				if failed {
					break
				}
				deleted := env.OnActivity(ia.Delete, mock.Anything, it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: tc.institution.ID}}).
					Return(it.DeleteInstitutionResult{}, tc.deleteErrs[n])
				if tc.deleteErrs[n] != assert.AnError {
					deleted.Once()
				}
			}

			env.ExecuteWorkflow(DeleteInstitution, tc.params)

			env.AssertExpectations(t)
			require.Equal(t, want, started)
			if tc.wantErr != "" {
				require.ErrorContains(t, env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(t, env.GetWorkflowError())
		})
	}
}
