package investments

import (
	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	pft "github.com/logicorg/ivfs/finance/common/types/portfoliotypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// vestCategories is what a vest books besides its units: the income category
// and the "Income" account of the vest's currency it is credited on, and the
// tax category with the "Expenses" account it is debited on, both zero when
// no units are withheld.
type vestCategories struct {
	income, incomeAccount int64
	tax, expensesAccount  int64
}

// retrieveVestCategories finds, for each category a vest names, the shared
// account of the kind it is booked as in the vest's currency.
func retrieveVestCategories(ctx workflow.Context, in *it.VestInput, currency *model.Currency) (*vestCategories, error) {
	ids := []int64{in.IncomeCategoryID}
	kinds := []model.LedgerAccountKind{model.LedgerAccountIncome}
	if in.UnitsWithheld.IsPositive() {
		ids = append(ids, in.TaxCategoryID)
		kinds = append(kinds, model.LedgerAccountExpense)
	}

	accounts := make([]int64, len(ids))
	for i, id := range ids {
		account, err := categorieswf.CategoryAccount(ctx, id, kinds[i], currency.ID)
		if err != nil {
			return nil, err
		}
		accounts[i] = account
	}

	result := &vestCategories{income: ids[0], incomeAccount: accounts[0]}
	if len(ids) > 1 {
		result.tax, result.expensesAccount = ids[1], accounts[1]
	}
	return result, nil
}

// buildVest is units received as compensation: "Income" is credited with the
// value of every unit vested, under the income category, "Investments"
// debited with that of the units delivered, and "Expenses" with that of the
// units withheld, under the tax category. The delivered units are added at their value. Each value is
// rounded to the currency's precision and the income is their sum, so the
// entry balances; a withholding worth less than the smallest unit is not
// posted, as a posting must be positive.
func buildVest(in *it.VestInput, b *brokerage, c *vestCategories) pft.RegisterVestInput {
	precision := int32(b.currency.Precision)
	delivered := in.UnitsVested.Sub(in.UnitsWithheld)
	deliveredValue := delivered.Mul(in.FairMarketValue).Round(precision)
	withheldValue := in.UnitsWithheld.Mul(in.FairMarketValue).Round(precision)

	postings := []jt.PostingInput{{LedgerAccountID: b.investments, Amount: deliveredValue, Side: model.PostingDebit}}
	if withheldValue.IsPositive() {
		postings = append(postings, jt.PostingInput{LedgerAccountID: c.expensesAccount, Amount: withheldValue, Side: model.PostingDebit, CategoryID: &c.tax})
	}
	postings = append(postings, jt.PostingInput{LedgerAccountID: c.incomeAccount, Amount: deliveredValue.Add(withheldValue), Side: model.PostingCredit, CategoryID: &c.income})

	result := pft.RegisterVestInput{
		Entry: jt.PostJournalEntryInput{
			IdempotencyKey: string(it.KindVest) + ":" + in.IdempotencyKey,
			OccurredAt:     in.OccurredAt,
			Postings:       postings,
			TagIDs:         in.TagIDs,
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
