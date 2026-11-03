package portfolio

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// vest is a vest of 100 units worth 10 each. The employer withholds 40 of them
// in shares, which leaves 60 units worth 600 for the account, and 50 more of tax
// is paid from cash. The entry books the whole 1000 as income.
func (s *UnitTestSuite) vest(key string) lt.RegisterVestInput {
	return lt.RegisterVestInput{
		FinancialAccountID:  s.financialAccount.ID,
		AssetID:             s.asset.ID,
		Units:               decimal.NewFromInt(60),
		CostBasisAmount:     decimal.NewFromInt(600),
		CostBasisCurrencyID: s.eur.ID,
		Entry: lt.PostJournalEntryInput{
			IdempotencyKey: key,
			OccurredAt:     movementDay,
			ExternalID:     ptr.NewPtr("payroll-42"),
			Postings: []lt.PostingInput{
				{LedgerAccountID: s.income.ID, Amount: decimal.NewFromInt(1000), Side: model.PostingCredit},
				{LedgerAccountID: s.holding.ID, Amount: decimal.NewFromInt(600), Side: model.PostingDebit},
				{LedgerAccountID: s.tax.ID, Amount: decimal.NewFromInt(400), Side: model.PostingDebit},
				{LedgerAccountID: s.tax.ID, Amount: decimal.NewFromInt(50), Side: model.PostingDebit},
				{LedgerAccountID: s.cash.ID, Amount: decimal.NewFromInt(50), Side: model.PostingCredit},
			},
		},
	}
}

func (s *UnitTestSuite) countEntriesAndPostings() (entries, postings int) {
	s.T().Helper()

	entries, err := s.DB.NewSelect().Model((*model.JournalEntry)(nil)).Count(s.T().Context())
	require.NoError(s.T(), err)
	postings, err = s.DB.NewSelect().Model((*model.Posting)(nil)).Count(s.T().Context())
	require.NoError(s.T(), err)
	return entries, postings
}

// Test_RegisterVest checks that a vest writes its entry, its postings, and the
// movement of the units received together, and that the movement is a vest with
// the cost basis and the date of the entry.
func (s *UnitTestSuite) Test_RegisterVest() {
	input := s.vest("acme-rsu-1")

	activityResult, err := s.Env.ExecuteActivity(s.pa.RegisterVest, input)

	require.NoError(s.T(), err)
	var result lt.RegisterVestResult
	require.NoError(s.T(), activityResult.Get(&result))

	var entry model.JournalEntry
	require.NoError(s.T(), s.DB.NewSelect().Model(&entry).Where("id = ?", result.JournalEntryID).Scan(s.T().Context()))
	require.Equal(s.T(), model.JournalEntryPosted, entry.Status)
	require.Equal(s.T(), "acme-rsu-1", *entry.IdempotencyKey)
	entries, postings := s.countEntriesAndPostings()
	require.Equal(s.T(), 1, entries)
	require.Equal(s.T(), len(input.Entry.Postings), postings)

	var movement model.PositionMovement
	require.NoError(s.T(), s.DB.NewSelect().Model(&movement).Where("pm.id = ?", result.PositionMovementID).Scan(s.T().Context()))
	require.Equal(s.T(), model.PositionMovementVest, movement.Kind)
	require.Equal(s.T(), result.JournalEntryID, *movement.JournalEntryID)
	require.Equal(s.T(), s.financialAccount.ID, movement.FinancialAccountID)
	require.Equal(s.T(), s.asset.ID, movement.AssetID)
	require.True(s.T(), movement.UnitsDelta.Equal(decimal.NewFromInt(60)))
	require.True(s.T(), movement.CostBasisAmount.Equal(decimal.NewFromInt(600)))
	require.Equal(s.T(), s.eur.ID, *movement.CostBasisCurrencyID)
	require.True(s.T(), movementDay.Equal(movement.OccurredAt))
	require.Equal(s.T(), "payroll-42", *movement.ExternalID)

	require.True(s.T(), s.unitsHeld(s.financialAccount, s.asset).Equal(decimal.NewFromInt(60)), "the units withheld were never held")
}

// Test_RegisterVest_Idempotent checks that a retry returns the entry and the
// movement of the first call and changes nothing, even when the retried request
// differs.
func (s *UnitTestSuite) Test_RegisterVest_Idempotent() {
	input := s.vest("acme-rsu-1")

	first, err := s.Env.ExecuteActivity(s.pa.RegisterVest, input)
	require.NoError(s.T(), err)
	var created lt.RegisterVestResult
	require.NoError(s.T(), first.Get(&created))
	entries, postings := s.countEntriesAndPostings()
	movements := s.countMovements()

	second, err := s.Env.ExecuteActivity(s.pa.RegisterVest, input)
	require.NoError(s.T(), err)
	var retried lt.RegisterVestResult
	require.NoError(s.T(), second.Get(&retried))

	require.Equal(s.T(), created, retried)
	afterEntries, afterPostings := s.countEntriesAndPostings()
	require.Equal(s.T(), entries, afterEntries)
	require.Equal(s.T(), postings, afterPostings)
	require.Equal(s.T(), movements, s.countMovements())
	require.True(s.T(), s.unitsHeld(s.financialAccount, s.asset).Equal(decimal.NewFromInt(60)))

	different := s.vest("acme-rsu-1")
	different.Units = decimal.NewFromInt(99)
	third, err := s.Env.ExecuteActivity(s.pa.RegisterVest, different)
	require.NoError(s.T(), err)
	var unchanged lt.RegisterVestResult
	require.NoError(s.T(), third.Get(&unchanged))

	require.Equal(s.T(), created, unchanged)
	require.Equal(s.T(), movements, s.countMovements())
	require.True(s.T(), s.unitsHeld(s.financialAccount, s.asset).Equal(decimal.NewFromInt(60)))
}

// Test_RegisterVest_Rejected checks that a vest that cannot be written leaves
// nothing behind, not even the entry and postings that were written before the
// failure was found.
func (s *UnitTestSuite) Test_RegisterVest_Rejected() {
	tests := map[string]struct {
		// input builds the vest to attempt, after seeding what the case needs.
		input func(key string) lt.RegisterVestInput
		// failure is what the error has to contain.
		failure string
	}{
		"a closed financial account": {
			failure: "is closed",
			input: func(key string) lt.RegisterVestInput {
				input := s.vest(key)
				input.FinancialAccountID = s.closedAccount("Closed " + key).ID
				return input
			},
		},
		"a financial account that does not exist": {
			failure: "FinancialAccount",
			input: func(key string) lt.RegisterVestInput {
				input := s.vest(key)
				input.FinancialAccountID = 999999
				return input
			},
		},
		"an asset that does not exist": {
			failure: "Asset",
			input: func(key string) lt.RegisterVestInput {
				input := s.vest(key)
				input.AssetID = 999999
				return input
			},
		},
		"a cost basis currency that does not exist": {
			failure: "Currency",
			input: func(key string) lt.RegisterVestInput {
				input := s.vest(key)
				input.CostBasisCurrencyID = 999999
				return input
			},
		},
		"an entry that does not balance": {
			failure: "balance",
			input: func(key string) lt.RegisterVestInput {
				input := s.vest(key)
				input.Entry.Postings[2].Amount = decimal.NewFromInt(300)
				return input
			},
		},
		"the key of a trade": {
			failure: "not a vest",
			input: func(key string) lt.RegisterVestInput {
				trade := &model.PositionMovement{FinancialAccountID: s.financialAccount.ID, AssetID: s.asset.ID, UnitsDelta: decimal.NewFromInt(5)}
				temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB,
					&model.JournalEntry{OccurredAt: movementDay, IdempotencyKey: ptr.NewPtr(key)},
					[]model.Posting{
						{LedgerAccountID: s.offset.ID, Amount: decimal.NewFromInt(50), Side: model.PostingDebit},
						{LedgerAccountID: s.cash.ID, Amount: decimal.NewFromInt(50), Side: model.PostingCredit},
					},
					[]*model.PositionMovement{trade})
				return s.vest(key)
			},
		},
		"the key of an entry without a movement": {
			failure: "without a vest",
			input: func(key string) lt.RegisterVestInput {
				temporaltesting.InsertPostedJournalEntry(s.T(), s.DB,
					&model.JournalEntry{OccurredAt: movementDay, IdempotencyKey: ptr.NewPtr(key)},
					model.Posting{LedgerAccountID: s.offset.ID, Amount: decimal.NewFromInt(50), Side: model.PostingDebit},
					model.Posting{LedgerAccountID: s.cash.ID, Amount: decimal.NewFromInt(50), Side: model.PostingCredit},
				)
				return s.vest(key)
			},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			input := tc.input(name)
			entries, postings := s.countEntriesAndPostings()
			movements := s.countMovements()

			_, err := s.Env.ExecuteActivity(s.pa.RegisterVest, input)

			require.ErrorContains(s.T(), err, tc.failure)
			afterEntries, afterPostings := s.countEntriesAndPostings()
			require.Equal(s.T(), entries, afterEntries, "a rejected vest must leave no entry behind")
			require.Equal(s.T(), postings, afterPostings, "a rejected vest must leave no postings behind")
			require.Equal(s.T(), movements, s.countMovements(), "a rejected vest must leave no movement behind")
		})
	}
}

type vestOutcome struct {
	result *lt.RegisterVestResult
	err    error
}

// Test_RegisterVest_RetryWaitsForConcurrentWriter checks that a retry that
// arrives while the first call is still in its transaction waits for it and
// returns its entry and movement, instead of missing them and colliding on the
// key.
func (s *UnitTestSuite) Test_RegisterVest_RetryWaitsForConcurrentWriter() {
	key := "racing-vest"
	writer, err := s.DB.BeginTx(s.T().Context(), nil)
	require.NoError(s.T(), err)
	s.T().Cleanup(func() { _ = writer.Rollback() })
	_, err = writer.ExecContext(s.T().Context(), "SELECT id FROM account.financial_account WHERE id = ? FOR NO KEY UPDATE", s.financialAccount.ID)
	require.NoError(s.T(), err)

	entry := &model.JournalEntry{OccurredAt: movementDay, Status: model.JournalEntryDraft, IdempotencyKey: ptr.NewPtr(key)}
	_, err = writer.NewInsert().Model(entry).Returning("id").Exec(s.T().Context())
	require.NoError(s.T(), err)
	movement := &model.PositionMovement{
		Kind: model.PositionMovementVest, JournalEntryID: &entry.ID, FinancialAccountID: s.financialAccount.ID,
		AssetID: s.asset.ID, UnitsDelta: decimal.NewFromInt(60), OccurredAt: movementDay,
	}
	_, err = writer.NewInsert().Model(movement).Returning("id").Exec(s.T().Context())
	require.NoError(s.T(), err)

	done := make(chan vestOutcome, 1)
	go func() {
		result, err := s.pa.RegisterVest(s.T().Context(), s.vest(key))
		done <- vestOutcome{result, err}
	}()
	select {
	case <-done:
		s.T().Fatal("the retry finished without waiting for the transaction holding the account")
	case <-time.After(500 * time.Millisecond):
	}

	require.NoError(s.T(), writer.Commit())
	outcome := <-done
	require.NoError(s.T(), outcome.err)
	require.Equal(s.T(), lt.RegisterVestResult{JournalEntryID: entry.ID, PositionMovementID: movement.ID}, *outcome.result)
	require.Equal(s.T(), 1, s.countMovements())
}
