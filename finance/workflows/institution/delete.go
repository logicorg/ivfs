package institution

import (
	"fmt"

	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"go.temporal.io/sdk/workflow"
)

// la is a (nil) pointer to the ledger account activity struct, for listing
// the institution's financial accounts.
var la = (*ledgeractivity.Activities)(nil)

// DeleteInstitution deletes an institution with every financial account
// under it, closed ones included, as ADR 0012 describes. Each account is
// deleted in turn as a DeleteFinancialAccount child workflow, so a transfer
// between two accounts of the same institution ends up deleted rather than
// kept. An institution is not merged. Deleting an institution that does not
// exist succeeds.
//
// It runs under DeleteInstitutionWorkflowID, and starts each child under
// DeleteFinancialAccountWorkflowID through ExecuteDeletionChild, which waits
// for an account whose deletion is already running as a separate request.
func DeleteInstitution(ctx workflow.Context, params it.DeleteInstitutionParams) error {
	if err := params.Validate(); err != nil {
		return err
	}
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	institution, err := retrieveInstitution(ctx, params.FindBy)
	if workflows.HasErrorType(err, "ErrNotFound") {
		return nil
	}
	if err != nil {
		return fmt.Errorf("retrieve institution: %w", err)
	}

	// A listing is capped, and an account can be opened meanwhile, so the
	// accounts are listed again until the institution's own row goes.
	for {
		var listed lt.ListFinancialAccountsResult
		list := lt.ListFinancialAccountsParams{InstitutionID: &institution.ID, IncludeClosed: true}
		if err := workflow.ExecuteActivity(ctx, la.ListFinancialAccounts, list).Get(ctx, &listed); err != nil {
			return fmt.Errorf("list financial accounts of institution %d: %w", institution.ID, err)
		}
		// The stray financial account lives under "Untracked", and deleting it
		// would leave nowhere to move a kept leg.
		if len(listed.FinancialAccounts) > 0 {
			if err := refuseUntrackedInstitution(institution, "deleted while it holds financial accounts"); err != nil {
				return err
			}
		}

		for _, account := range listed.FinancialAccounts {
			input := lt.DeleteFinancialAccountInput{FinancialAccountID: account.ID}
			if err := workflows.ExecuteDeletionChild(ctx, workflows.DeleteFinancialAccountWorkflowID(account.ID),
				ledgeraccountswf.DeleteFinancialAccount, input, nil); err != nil {
				return fmt.Errorf("delete financial account %d of institution %d: %w", account.ID, institution.ID, err)
			}
		}

		err := workflow.ExecuteActivity(ctx, ia.Delete, it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: institution.ID}}).Get(ctx, nil)
		if err == nil {
			return nil
		}
		if !workflows.HasErrorType(err, "ErrResourceConflict") || len(listed.FinancialAccounts) == 0 {
			return fmt.Errorf("delete institution: %w", err)
		}
	}
}
