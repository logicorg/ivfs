package accountsetup

import (
	"fmt"

	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// AddBalance adds one named asset balance, such as a savings vault, to an
// existing financial account, with the shared accounts of its currency, and
// books its opening balance (ADR 0006). A balance with the same name and
// currency in the account is returned unchanged, and a repeated request posts
// no second opening entry. A closed account or institution fails with
// ErrResourceClosed before any write.
func AddBalance(ctx workflow.Context, input ast.AddBalanceInput) (*ast.AddBalanceResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	currency, err := retrieveCurrency(ctx, input.CurrencyCode)
	if err != nil {
		return nil, err
	}
	account, err := retrieveOpenFinancialAccount(ctx, input.FinancialAccountID)
	if err != nil {
		return nil, err
	}

	at := input.OpeningBalanceAt
	if at.IsZero() {
		at = workflow.Now(ctx)
	}
	balances, err := openBalances(ctx, *account, currency,
		[]balanceSpec{{input.Name, model.LedgerAccountAsset}}, input.OpeningBalance, at)
	if err != nil {
		return nil, err
	}

	return &ast.AddBalanceResult{
		Outcome:         ast.OutcomeOpened,
		LedgerAccountID: balances.Balances[0].LedgerAccountID,
		Balances:        *balances,
	}, nil
}

// retrieveOpenFinancialAccount retrieves a financial account and its
// institution, failing with ErrResourceClosed when either is closed.
func retrieveOpenFinancialAccount(ctx workflow.Context, id int64) (*financialAccountRef, error) {
	var account lt.RetrieveFinancialAccountResult
	err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.RetrieveFinancialAccount, lt.RetrieveFinancialAccountParams{ID: id}).
		Get(ctx, &account)
	if err != nil {
		return nil, fmt.Errorf("retrieve financial account %d: %w", id, err)
	}
	if account.FinancialAccount.ClosedAt != nil {
		return nil, internalerrors.ErrResourceClosed{ResourceType: "FinancialAccount", ID: id}
	}

	var institution it.RetrieveInstitutionResult
	err = workflow.ExecuteChildWorkflow(ctx, institutionwf.RetrieveInstitution, it.RetrieveInstitutionParams{
		ID: account.FinancialAccount.InstitutionID,
	}).Get(ctx, &institution)
	if err != nil {
		return nil, fmt.Errorf("retrieve the institution of financial account %d: %w", id, err)
	}
	if institution.Institution.ClosedAt != nil {
		return nil, internalerrors.ErrResourceClosed{ResourceType: "Institution", ID: institution.Institution.ID}
	}

	return &financialAccountRef{
		id: id, name: account.FinancialAccount.Name, institutionName: institution.Institution.Name,
	}, nil
}
