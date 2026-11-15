package investments

import (
	"fmt"

	catt "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
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

// retrieveVestCategories finds the categories a vest names, checks each is of
// the kind it is booked as, and finds the shared account of that kind in the
// vest's currency. A category has no currency and may be closed.
func retrieveVestCategories(ctx workflow.Context, in *it.VestInput, currency *model.Currency) (*vestCategories, error) {
	ids := []int64{in.IncomeCategoryID}
	kinds := []model.LedgerAccountKind{model.LedgerAccountIncome}
	if in.UnitsWithheld.IsPositive() {
		ids = append(ids, in.TaxCategoryID)
		kinds = append(kinds, model.LedgerAccountExpense)
	}

	futures := make([]workflow.ChildWorkflowFuture, len(ids))
	for i, id := range ids {
		futures[i] = workflow.ExecuteChildWorkflow(ctx, categorieswf.RetrieveCategory, catt.RetrieveCategoryParams{ID: id})
	}
	accounts := make([]int64, len(ids))
	for i, future := range futures {
		var retrieved catt.RetrieveCategoryResult
		if err := future.Get(ctx, &retrieved); err != nil {
			return nil, fmt.Errorf("retrieve category %d: %w", ids[i], err)
		}
		if err := catt.CheckKind(retrieved.Category, kinds[i]); err != nil {
			return nil, err
		}
		account, err := ledgeraccountswf.EnsureCategoryAccount(ctx, kinds[i], currency.ID)
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
func buildVest(in *it.VestInput, b *brokerage, c *vestCategories) lt.RegisterVestInput {
	precision := int32(b.currency.Precision)
	delivered := in.UnitsVested.Sub(in.UnitsWithheld)
	deliveredValue := delivered.Mul(in.FairMarketValue).Round(precision)
	withheldValue := in.UnitsWithheld.Mul(in.FairMarketValue).Round(precision)

	postings := []lt.PostingInput{{LedgerAccountID: b.investments, Amount: deliveredValue, Side: model.PostingDebit}}
	if withheldValue.IsPositive() {
		postings = append(postings, lt.PostingInput{LedgerAccountID: c.expensesAccount, Amount: withheldValue, Side: model.PostingDebit, CategoryID: &c.tax})
	}
	postings = append(postings, lt.PostingInput{LedgerAccountID: c.incomeAccount, Amount: deliveredValue.Add(withheldValue), Side: model.PostingCredit, CategoryID: &c.income})

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
