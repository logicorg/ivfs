package accounts

import (
	"fmt"

	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

var (
	// la is a (nil) pointer to the ledger accounts activities struct. It is used to call the activities in the workflow.
	la = (*ledgeractivity.Activities)(nil)
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

func CreateCounterparty(ctx workflow.Context, input lt.CreateCounterpartyInput) (*lt.CreateCounterpartyResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.CreateCounterpartyResult
	if err := workflow.ExecuteActivity(ctx, la.CreateCounterparty, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("create counterparty: %w", err)
	}
	return &result, nil
}

// RetrieveCounterparty is a workflow that retrieves a counterparty.
func RetrieveCounterparty(ctx workflow.Context, params lt.RetrieveCounterpartyParams) (*lt.RetrieveCounterpartyResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.RetrieveCounterpartyResult
	if err := workflow.ExecuteActivity(ctx, la.RetrieveCounterparty, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve counterparty: %w", err)
	}
	return &result, nil
}

// ListCounterparties is a workflow that lists/searches counterparties.
func ListCounterparties(ctx workflow.Context, params lt.ListCounterpartiesParams) (*lt.ListCounterpartiesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.ListCounterpartiesResult
	if err := workflow.ExecuteActivity(ctx, la.ListCounterparties, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list counterparties: %w", err)
	}
	return &result, nil
}

// RenameCounterparty is a workflow that renames a counterparty.
func RenameCounterparty(ctx workflow.Context, input lt.RenameCounterpartyInput) (*lt.CounterpartyMutationResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.CounterpartyMutationResult
	if err := workflow.ExecuteActivity(ctx, la.RenameCounterparty, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("rename counterparty: %w", err)
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
