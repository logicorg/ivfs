package journal

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
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

	posted, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{
		IdempotencyKey: "retrieve-entry", OccurredAt: time.Now().UTC(),
		Postings: []lt.PostingInput{{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(12), Side: model.PostingDebit}, {LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(12), Side: model.PostingCredit}},
	})
	require.NoError(s.T(), err)
	var postedResult lt.PostJournalEntryResult
	require.NoError(s.T(), posted.Get(&postedResult))

	activityResult, err := s.Env.ExecuteActivity(s.ja.RetrieveJournalEntry, lt.RetrieveJournalEntryParams{ID: postedResult.JournalEntryID})
	require.NoError(s.T(), err)
	var result lt.RetrieveJournalEntryResult
	require.NoError(s.T(), activityResult.Get(&result))
	require.Equal(s.T(), postedResult.JournalEntryID, result.JournalEntry.ID)
	require.Equal(s.T(), model.JournalEntryPosted, result.JournalEntry.Status)
	require.Len(s.T(), result.JournalEntry.Postings, 2)
}
