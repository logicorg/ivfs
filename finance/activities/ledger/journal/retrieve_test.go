package journal

import (
	"time"

	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) TestRetrieveJournalEntryWithRelationships() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)

	debit := testLedgerAccount(s.T(), s.DB, euro.ID, "cash", model.LedgerAccountAsset)
	credit := testLedgerAccount(s.T(), s.DB, euro.ID, "expense", model.LedgerAccountExpense)

	refunds := &model.Category{Kind: model.LedgerAccountExpense, Name: "Refunds"}
	temporaltesting.Insert(s.T(), s.DB, refunds)
	vacation, reimbursable := &model.Tag{Name: "vacation"}, &model.Tag{Name: "Reimbursable"}
	temporaltesting.Insert(s.T(), s.DB, vacation)
	temporaltesting.Insert(s.T(), s.DB, reimbursable)

	posted, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, jt.PostJournalEntryInput{
		IdempotencyKey: "retrieve-entry", OccurredAt: time.Now().UTC(), TagIDs: []int64{vacation.ID, reimbursable.ID},
		Postings: []jt.PostingInput{
			{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(12), Side: model.PostingDebit},
			{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(12), Side: model.PostingCredit, CategoryID: &refunds.ID},
		},
	})
	require.NoError(s.T(), err)
	var postedResult jt.PostJournalEntryResult
	require.NoError(s.T(), posted.Get(&postedResult))

	activityResult, err := s.Env.ExecuteActivity(s.ja.RetrieveJournalEntry, jt.RetrieveJournalEntryParams{ID: postedResult.JournalEntryID})
	require.NoError(s.T(), err)
	var result jt.RetrieveJournalEntryResult
	require.NoError(s.T(), activityResult.Get(&result))
	require.Equal(s.T(), postedResult.JournalEntryID, result.JournalEntry.ID)
	require.Equal(s.T(), model.JournalEntryPosted, result.JournalEntry.Status)
	require.Len(s.T(), result.JournalEntry.Postings, 2)

	uncategorized, categorized := result.JournalEntry.Postings[0], result.JournalEntry.Postings[1]
	require.Nil(s.T(), uncategorized.CategoryID)
	require.Nil(s.T(), uncategorized.Category, "a posting without a category loads none")
	require.Equal(s.T(), refunds.ID, *categorized.CategoryID)
	require.NotNil(s.T(), categorized.Category)
	require.Equal(s.T(), "Refunds", categorized.Category.Name)

	require.Len(s.T(), result.JournalEntry.Tags, 2)
	require.Equal(s.T(), "Reimbursable", result.JournalEntry.Tags[0].Name, "tags are ordered by name whatever the casing")
	require.Equal(s.T(), vacation.ID, result.JournalEntry.Tags[1].ID)
}
