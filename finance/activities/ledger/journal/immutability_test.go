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

func (s *UnitTestSuite) TestPostedPostingCannotBeMutated() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)

	debit := testLedgerAccount(s.T(), s.DB, euro.ID, "debit", model.LedgerAccountAsset)
	credit := testLedgerAccount(s.T(), s.DB, euro.ID, "credit", model.LedgerAccountExpense)

	posted, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{IdempotencyKey: "immutable-entry", OccurredAt: time.Now().UTC(), Postings: []lt.PostingInput{
		{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(1), Side: model.PostingDebit},
		{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(1), Side: model.PostingCredit},
	}})

	require.NoError(s.T(), err)
	var result lt.PostJournalEntryResult
	require.NoError(s.T(), posted.Get(&result))

	var posting model.Posting
	require.NoError(s.T(), s.DB.NewSelect().Model(&posting).Where("journal_entry_id = ?", result.JournalEntryID).Order("line_number").Limit(1).Scan(s.T().Context()))
	posting.Amount = decimal.NewFromInt(2)
	_, err = s.DB.NewUpdate().Model(&posting).WherePK().Exec(s.T().Context())
	require.Error(s.T(), err)
	_, err = s.DB.NewDelete().Model(&posting).WherePK().Exec(s.T().Context())
	require.Error(s.T(), err)
}
