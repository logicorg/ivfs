package accounts

import (
	"fmt"

	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	accountdeletion "github.com/logicorg/ivfs/finance/activities/ledger/accounts/deletion"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

var (
	// la is a (nil) pointer to the ledger accounts activities struct. It is used to call the activities in the workflow.
	la = (*ledgeractivity.Activities)(nil)
	// da is a (nil) pointer to the account deletion activities struct, for the
	// activities that close, empty and preview an account being deleted.
	da = (*accountdeletion.Activities)(nil)
	// ia is a (nil) pointer to the institution activities struct, for the
	// checks that need a financial account's institution.
	ia = (*institutionactivity.Institution)(nil)
)

// RetrieveFinancialAccount is a workflow that retrieves a financial account.
func RetrieveFinancialAccount(ctx workflow.Context, params lt.RetrieveFinancialAccountParams) (*lt.RetrieveFinancialAccountResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.RetrieveFinancialAccountResult
	if err := workflow.ExecuteActivity(ctx, la.RetrieveFinancialAccount, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve financial account: %w", err)
	}
	return &result, nil
}

// ListFinancialAccounts is a workflow that lists financial accounts.
func ListFinancialAccounts(ctx workflow.Context, params lt.ListFinancialAccountsParams) (*lt.ListFinancialAccountsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.ListFinancialAccountsResult
	if err := workflow.ExecuteActivity(ctx, la.ListFinancialAccounts, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list financial accounts: %w", err)
	}
	return &result, nil
}

func CreateLedgerAccount(ctx workflow.Context, input lt.CreateLedgerAccountInput) (*lt.CreateLedgerAccountResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.CreateLedgerAccountResult
	if err := workflow.ExecuteActivity(ctx, la.CreateLedgerAccount, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("create ledger account: %w", err)
	}
	return &result, nil
}

func RetrieveLedgerAccount(ctx workflow.Context, params lt.RetrieveLedgerAccountParams) (*lt.RetrieveLedgerAccountResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.RetrieveLedgerAccountResult
	if err := workflow.ExecuteActivity(ctx, la.RetrieveLedgerAccount, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve ledger account: %w", err)
	}
	return &result, nil
}

func ListLedgerAccounts(ctx workflow.Context, params lt.ListLedgerAccountsParams) (*lt.ListLedgerAccountsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.ListLedgerAccountsResult
	if err := workflow.ExecuteActivity(ctx, la.ListLedgerAccounts, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list ledger accounts: %w", err)
	}
	return &result, nil
}
