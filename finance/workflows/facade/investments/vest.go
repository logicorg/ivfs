package investments

import (
	"fmt"

	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// vestCategories is the income category a vest credits and the expense one
// it debits for the units withheld, zero when none are.
type vestCategories struct {
	income, tax int64
}

// retrieveVestCategories finds the categories a vest names and checks each is
// of the kind it is booked as and in the vest's currency, which the ledger
// would otherwise refuse only as an unbalanced entry naming currency ids.
func retrieveVestCategories(ctx workflow.Context, in *it.VestInput, currency *model.Currency) (*vestCategories, error) {
	ids := []int64{in.IncomeCategoryLedgerAccountID}
	kinds := []model.LedgerAccountKind{model.LedgerAccountIncome}
	if in.UnitsWithheld.IsPositive() {
		ids = append(ids, in.TaxCategoryLedgerAccountID)
		kinds = append(kinds, model.LedgerAccountExpense)
	}

	accounts, err := retrieveLedgerAccounts(ctx, ids...)
	if err != nil {
		return nil, err
	}
	for i, account := range accounts {
		if err := checkCategory(account, kinds[i], currency); err != nil {
			return nil, err
		}
	}

	result := &vestCategories{income: accounts[0].ID}
	if len(accounts) > 1 {
		result.tax = accounts[1].ID
	}
	return result, nil
}

// checkCategory fails unless the account is a category of the given kind in
// the currency.
func checkCategory(account *model.LedgerAccount, kind model.LedgerAccountKind, currency *model.Currency) error {
	switch {
	case account.Kind == kind && account.CurrencyID == currency.ID:
		return nil
	case account.Kind == kind:
		return internalerrors.ErrInvalidParameter{
			Description: fmt.Sprintf("%q is not in %s, the currency of the vest: pick a category in %s", account.Name, currency.Code, currency.Code),
		}
	case account.Kind == model.LedgerAccountIncome || account.Kind == model.LedgerAccountExpense:
		return internalerrors.ErrInvalidParameter{
			Description: fmt.Sprintf("%q is an %s category, not an %s one", account.Name, account.Kind, kind),
		}
	default:
		return internalerrors.ErrInvalidParameter{
			Description: fmt.Sprintf("%q is not an %s category: its kind is %s", account.Name, kind, account.Kind),
		}
	}
}

// buildVest is units received as compensation: the income category is
// credited with the value of every unit vested, "Investments" debited with
// that of the units delivered, and the tax category with that of the units
// withheld. The delivered units are added at their value. Each value is
// rounded to the currency's precision and the income is their sum, so the
// entry balances; a withholding worth less than the smallest unit is not
// posted, as a posting must be positive.
func buildVest(in *it.VestInput, b *brokerage, c *vestCategories) lt.RegisterVestInput {
	precision := int32(b.currency.Precision)
	delivered := in.UnitsVested.Sub(in.UnitsWithheld)
	deliveredValue := delivered.Mul(in.FairMarketValue).Round(precision)
	withheldValue := in.UnitsWithheld.Mul(in.FairMarketValue).Round(precision)

	postings := []lt.PostingInput{{LedgerAccountID: b.investments, Amount: deliveredValue, Side: model.PostingDebit}}
	if withheldValue.IsPositive() {
		postings = append(postings, lt.PostingInput{LedgerAccountID: c.tax, Amount: withheldValue, Side: model.PostingDebit})
	}
	postings = append(postings, lt.PostingInput{LedgerAccountID: c.income, Amount: deliveredValue.Add(withheldValue), Side: model.PostingCredit})

	result := lt.RegisterVestInput{
		Entry: lt.PostJournalEntryInput{
			IdempotencyKey: string(it.KindVest) + ":" + in.IdempotencyKey,
			OccurredAt:     in.OccurredAt,
			Postings:       postings,
		},
		FinancialAccountID:  in.FinancialAccountID,
		AssetID:             in.AssetID,
		Units:               delivered,
		CostBasisAmount:     deliveredValue,
		CostBasisCurrencyID: b.currency.ID,
	}
	if in.Description != "" {
		result.Entry.Description = &in.Description
	}
	return result
}
