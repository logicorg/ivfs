package accountsetup

import (
	"fmt"

	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// OpenAccount opens a financial account from the answers to the account form
// (ADR 0006): it finds or creates the institution and the account, and gives
// the account the balances its type needs in each currency, with the shared
// accounts of each currency and their opening balances. Opening an account
// that exists adds the currencies it does not hold yet, and a repeated request
// changes nothing.
//
// A missing institution is created only when the request confirms it, or for
// a cash account; otherwise the outcome is institution_not_found and nothing
// is written. Every currency is resolved before any write, and a closed
// institution or account fails with ErrResourceClosed.
func OpenAccount(ctx workflow.Context, input ast.OpenAccountInput) (*ast.OpenAccountResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	currencies := make([]*model.Currency, 0, len(input.Currencies))
	for _, currency := range input.Currencies {
		resolved, err := retrieveCurrency(ctx, currency.Code)
		if err != nil {
			return nil, err
		}
		currencies = append(currencies, resolved)
	}

	name := institutionName(input)
	institution, err := retrieveInstitution(ctx, name)
	if err != nil {
		return nil, err
	}
	if institution == nil {
		if input.Type != ast.AccountTypeCash && !input.CreateInstitution {
			return &ast.OpenAccountResult{Outcome: ast.OutcomeInstitutionNotFound, InstitutionName: name}, nil
		}
		if institution, err = insertInstitution(ctx, name, institutionKind(input.Type)); err != nil {
			return nil, err
		}
	}

	var created lt.CreateFinancialAccountResult
	err = workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.CreateFinancialAccount, lt.CreateFinancialAccountInput{
		InstitutionID: institution.ID, Name: accountName(input),
	}).Get(ctx, &created)
	if err != nil {
		return nil, fmt.Errorf("create financial account %s: %w", accountName(input), err)
	}
	account, err := retrieveOpenFinancialAccount(ctx, created.FinancialAccountID)
	if err != nil {
		return nil, err
	}

	at := input.OpeningBalancesAt
	if at.IsZero() {
		at = workflow.Now(ctx)
	}
	result := &ast.OpenAccountResult{
		Outcome:              ast.OutcomeOpened,
		InstitutionID:        institution.ID,
		InstitutionName:      account.institutionName,
		FinancialAccountID:   account.id,
		FinancialAccountName: account.name,
	}
	for i, currency := range currencies {
		balances, err := openBalances(ctx, *account, currency, scaffold(input.Type), input.Currencies[i].OpeningBalance, at)
		if err != nil {
			return nil, err
		}
		result.Balances.Balances = append(result.Balances.Balances, balances.Balances...)
		result.OpeningEntries = append(result.OpeningEntries, balances.OpeningEntries...)
	}
	return result, nil
}

// retrieveInstitution finds an institution by name, or returns nil when there
// is none. A closed institution fails with ErrResourceClosed.
func retrieveInstitution(ctx workflow.Context, name string) (*model.Institution, error) {
	var retrieved it.RetrieveInstitutionResult
	err := workflow.ExecuteChildWorkflow(ctx, institutionwf.RetrieveInstitution, it.RetrieveInstitutionParams{Name: name}).
		Get(ctx, &retrieved)
	switch {
	case workflows.HasErrorType(err, "ErrNotFound"):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("retrieve institution %s: %w", name, err)
	case retrieved.Institution.ClosedAt != nil:
		return nil, internalerrors.ErrResourceClosed{ResourceType: "Institution", ID: retrieved.Institution.ID}
	}
	return retrieved.Institution, nil
}

// insertInstitution creates an institution, or finds the one with its name.
func insertInstitution(ctx workflow.Context, name, kind string) (*model.Institution, error) {
	var inserted it.InsertInstitutionResult
	err := workflow.ExecuteChildWorkflow(ctx, institutionwf.InsertInstitution, it.InsertInstitutionParams{Name: name, Kind: kind}).
		Get(ctx, &inserted)
	if err != nil {
		return nil, fmt.Errorf("insert institution %s: %w", name, err)
	}
	return &model.Institution{ID: inserted.ID, Name: name, Kind: kind}, nil
}
