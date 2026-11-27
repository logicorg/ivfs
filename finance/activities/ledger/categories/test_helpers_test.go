package categories

import (
	"context"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// missingID is the id of no category.
const missingID int64 = 999999

// closedAt is when the closed categories of the fixture were closed.
var closedAt = time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

// fixture is the categories the tests act on, by the name a test case gives
// them:
//
//	food ── fruit ── citrus     expense, three levels
//	home ── kitchen             expense, two levels
//	bills ── power              expense; power has one entry
//	attic ── dusty              expense; dusty is closed
//	travel                      expense
//	rent                        expense, with two entries
//	old                         expense, closed
//	cooking                     expense, named "Kitchen" like home's child
//	jobs                        income
//	wages                       income, with one entry
//
// "missing" is the id of no category.
type fixture map[string]*model.Category

// id returns the id of the category a test case names, and nil for "".
func (f fixture) id(name string) *int64 {
	switch name {
	case "":
		return nil
	case "missing":
		id := missingID
		return &id
	default:
		return &f[name].ID
	}
}

// seed inserts the fixture. A case that uses it clears it when it is done, so
// the next one starts from the same categories.
func (s *UnitTestSuite) seed() fixture {
	s.T().Helper()

	f := fixture{}
	add := func(key, name string, kind model.LedgerAccountKind, parent string, closed bool) {
		category := &model.Category{Name: name, Kind: kind, ParentID: f.id(parent)}
		if closed {
			category.ClosedAt = &closedAt
		}
		temporaltesting.Insert(s.T(), s.DB, category)
		f[key] = category
	}
	expense, income := model.LedgerAccountExpense, model.LedgerAccountIncome
	add("food", "Food", expense, "", false)
	add("fruit", "Fruit", expense, "food", false)
	add("citrus", "Citrus", expense, "fruit", false)
	add("home", "Home", expense, "", false)
	add("kitchen", "Kitchen", expense, "home", false)
	add("bills", "Bills", expense, "", false)
	add("power", "Power", expense, "bills", false)
	add("attic", "Attic", expense, "", false)
	add("dusty", "Dusty", expense, "attic", true)
	add("travel", "Travel", expense, "", false)
	add("rent", "Rent", expense, "", false)
	add("old", "Old", expense, "", true)
	add("cooking", "Kitchen", expense, "", false)
	add("jobs", "Jobs", income, "", false)
	add("wages", "Wages", income, "", false)

	eur := testhelpers.Euro()
	temporaltesting.InsertOrRetrieve(s.T(), s.DB, eur, "code")
	account := func(name string, kind model.LedgerAccountKind) *model.LedgerAccount {
		ledgerAccount := &model.LedgerAccount{CurrencyID: eur.ID, Name: name, Kind: kind}
		temporaltesting.Insert(s.T(), s.DB, ledgerAccount)
		return ledgerAccount
	}
	cash, expenses, earned := account("cash", model.LedgerAccountAsset), account("Expenses", expense), account("Income", income)

	spend := func(categories ...*model.Category) {
		postings := make([]model.Posting, 0, len(categories)+1)
		for _, category := range categories {
			postings = append(postings, model.Posting{LedgerAccountID: expenses.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit, CategoryID: &category.ID})
		}
		postings = append(postings, model.Posting{LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(int64(10 * len(categories))), Side: model.PostingCredit})
		temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, &model.JournalEntry{OccurredAt: closedAt}, postings...)
	}
	spend(f["power"])
	spend(f["rent"])
	// Two postings of one entry, which is still one entry of the category.
	spend(f["rent"], f["rent"])
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, &model.JournalEntry{OccurredAt: closedAt},
		model.Posting{LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(100), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: earned.ID, Amount: decimal.NewFromInt(100), Side: model.PostingCredit, CategoryID: &f["wages"].ID},
	)
	return f
}

// clear deletes everything seed inserted, and what a case added to it. The
// entries go with their postings in one transaction, and the categories from
// the leaves up, since a parent that is still referenced cannot be deleted.
func (s *UnitTestSuite) clear() {
	s.T().Helper()

	err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		statements := []string{
			`DELETE FROM ledger.posting`,
			`DELETE FROM ledger.journal_entry`,
			`DELETE FROM ledger.ledger_account`,
		}
		for range maxDepth + 1 {
			statements = append(statements, `DELETE FROM ledger.category AS c WHERE NOT EXISTS (SELECT 1 FROM ledger.category AS k WHERE k.parent_id = c.id)`)
		}
		for _, statement := range statements {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(s.T(), err)
}

// stored returns the category as the database has it, or nil when it is gone.
func (s *UnitTestSuite) stored(id int64) *model.Category {
	s.T().Helper()

	var categories []model.Category
	require.NoError(s.T(), s.DB.NewSelect().Model(&categories).Where("cat.id = ?", id).Scan(s.T().Context()))
	if len(categories) == 0 {
		return nil
	}
	return &categories[0]
}
