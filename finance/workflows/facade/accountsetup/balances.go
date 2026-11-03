package accountsetup

import (
	"fmt"
	"strings"
	"time"

	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/workflow"
)

// The steps OpenAccount and AddBalance share. Each runs a lower-level workflow
// as a child, and each is retrieve-or-insert, so running them again changes
// nothing.

// openingBalanceDescription describes every opening balance entry.
const openingBalanceDescription = "Opening balance"

// financialAccountRef is the financial account balances are opened in, with
// the names the result rows show.
type financialAccountRef struct {
	id              int64
	name            string
	institutionName string
}

// retrieveCurrency resolves a currency code, upper-cased as every code is
// stored (ADR 0005). An unknown code fails, with ErrNotFound in the error's
// cause chain.
func retrieveCurrency(ctx workflow.Context, code string) (*model.Currency, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	var currency model.Currency
	err := workflow.ExecuteChildWorkflow(ctx, currencieswf.RetrieveCurrency, &ct.RetrieveCurrencyParams{Code: code}).Get(ctx, &currency)
	if err != nil {
		return nil, fmt.Errorf("retrieve currency %s: %w", code, err)
	}
	return &currency, nil
}

// openBalances creates the balances in specs under account, in currency, and
// the shared accounts of currency, then books opening on the first of specs.
// The opening entry's idempotency key is derived from its balance, so a
// balance has at most one opening entry however often this runs.
func openBalances(
	ctx workflow.Context, account financialAccountRef, currency *model.Currency,
	specs []balanceSpec, opening decimal.Decimal, at time.Time,
) (*ast.Balances, error) {
	result := &ast.Balances{}

	balances := make([]*model.LedgerAccount, 0, len(specs))
	for _, spec := range specs {
		balance, err := ensureLedgerAccount(ctx, lt.CreateLedgerAccountInput{
			FinancialAccountID: &account.id, CurrencyID: currency.ID, Name: spec.name, Kind: spec.kind,
		})
		if err != nil {
			return nil, err
		}
		balances = append(balances, balance)
		result.Balances = append(result.Balances, summarise(balance, &account))
	}

	shared := make([]*model.LedgerAccount, 0, len(sharedAccounts()))
	for _, spec := range sharedAccounts() {
		sharedAccount, err := ensureLedgerAccount(ctx, lt.CreateLedgerAccountInput{
			CurrencyID: currency.ID, Name: spec.name, Kind: spec.kind,
		})
		if err != nil {
			return nil, err
		}
		shared = append(shared, sharedAccount)
		result.Balances = append(result.Balances, summarise(sharedAccount, nil))
	}

	postings := openingBalancePostings(balances[0].ID, balances[0].Kind, shared[0].ID, opening)
	if postings == nil {
		return result, nil
	}
	var posted lt.PostJournalEntryResult
	err := workflow.ExecuteChildWorkflow(ctx, journalwf.PostJournalEntry, lt.PostJournalEntryInput{
		IdempotencyKey: openingBalanceKey(balances[0].ID),
		OccurredAt:     at,
		Description:    ptr.NewPtr(openingBalanceDescription),
		Postings:       postings,
	}).Get(ctx, &posted)
	if err != nil {
		return nil, fmt.Errorf("post the opening balance of %s: %w", balances[0].Name, err)
	}
	result.OpeningEntries = append(result.OpeningEntries, ast.OpeningEntry{
		LedgerAccountID: balances[0].ID, JournalEntryID: posted.JournalEntryID,
	})
	return result, nil
}

// ensureLedgerAccount creates a ledger account, or finds the one with its key,
// and retrieves it, so the result shows it as stored: in the casing it was
// first given, and closed if it is.
func ensureLedgerAccount(ctx workflow.Context, input lt.CreateLedgerAccountInput) (*model.LedgerAccount, error) {
	var created lt.CreateLedgerAccountResult
	if err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.CreateLedgerAccount, input).Get(ctx, &created); err != nil {
		return nil, fmt.Errorf("create ledger account %s: %w", input.Name, err)
	}

	var retrieved lt.RetrieveLedgerAccountResult
	err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{
		ID: created.LedgerAccountID,
	}).Get(ctx, &retrieved)
	if err != nil {
		return nil, fmt.Errorf("retrieve ledger account %s: %w", input.Name, err)
	}
	return retrieved.LedgerAccount, nil
}

// summarise is a ledger account as a row of the result, under account, or
// under no financial account when account is nil.
func summarise(ledgerAccount *model.LedgerAccount, account *financialAccountRef) lt.AccountSummary {
	summary := lt.AccountSummary{
		LedgerAccountID:   ledgerAccount.ID,
		LedgerAccountName: ledgerAccount.Name,
		Kind:              ledgerAccount.Kind,
		ParentID:          ledgerAccount.ParentID,
		CurrencyID:        ledgerAccount.CurrencyID,
		Closed:            ledgerAccount.ClosedAt != nil,
	}
	if ledgerAccount.Currency != nil {
		summary.CurrencyCode = ledgerAccount.Currency.Code
	}
	if account != nil {
		summary.FinancialAccountID = &account.id
		summary.FinancialAccountName = account.name
		summary.InstitutionName = account.institutionName
	}
	return summary
}
