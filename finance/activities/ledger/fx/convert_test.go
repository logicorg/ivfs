package fx

import (
	"fmt"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) execute(input lt.RegisterFXConversionInput) (lt.RegisterFXConversionResult, error) {
	s.T().Helper()

	encoded, err := s.Env.ExecuteActivity(s.fx.RegisterFXConversion, input)
	if err != nil {
		return lt.RegisterFXConversionResult{}, err
	}

	var result lt.RegisterFXConversionResult
	require.NoError(s.T(), encoded.Get(&result))
	return result, nil
}

func (s *UnitTestSuite) postingsOf(entryID int64) []string {
	s.T().Helper()

	var postings []model.Posting
	require.NoError(s.T(), s.DB.NewSelect().Model(&postings).Where("journal_entry_id = ?", entryID).Scan(s.T().Context()))

	lines := make([]string, 0, len(postings))
	for _, p := range postings {
		lines = append(lines, fmt.Sprintf("%s account %d %s", p.Side, p.LedgerAccountID, p.Amount))
	}
	return lines
}

func (s *UnitTestSuite) Test_RegisterFXConversion() {
	input := s.adrExample()
	input.ProviderReference = ptr.NewPtr("revolut-tx-42")

	result, err := s.execute(input)
	require.NoError(s.T(), err)

	require.ElementsMatch(s.T(), []string{
		fmt.Sprintf("debit account %d 99", s.usdClearing.ID),
		fmt.Sprintf("debit account %d 1", s.usdFees.ID),
		fmt.Sprintf("credit account %d 100", s.revolutUSD.ID),
		fmt.Sprintf("debit account %d 90", s.revolutEUR.ID),
		fmt.Sprintf("credit account %d 90", s.eurClearing.ID),
	}, s.postingsOf(result.JournalEntryID))

	var entry model.JournalEntry
	require.NoError(s.T(), s.DB.NewSelect().Model(&entry).Where("id = ?", result.JournalEntryID).Scan(s.T().Context()))
	require.Equal(s.T(), model.JournalEntryPosted, entry.Status)
	require.Equal(s.T(), map[string]string{
		"type":               lt.FXConversionEntryType,
		"effective_rate":     "90/99",
		"provider_reference": "revolut-tx-42",
	}, entry.Metadata)
}

func (s *UnitTestSuite) Test_RegisterFXConversion_WithoutFee() {
	input := s.adrExample()
	input.Fee = nil

	result, err := s.execute(input)
	require.NoError(s.T(), err)

	require.ElementsMatch(s.T(), []string{
		fmt.Sprintf("debit account %d 100", s.usdClearing.ID),
		fmt.Sprintf("credit account %d 100", s.revolutUSD.ID),
		fmt.Sprintf("debit account %d 90", s.revolutEUR.ID),
		fmt.Sprintf("credit account %d 90", s.eurClearing.ID),
	}, s.postingsOf(result.JournalEntryID))
}

// Retrying must return the original entry even if the retried request differs,
// following the retrieve-or-insert convention for creation activities.
func (s *UnitTestSuite) Test_RegisterFXConversion_Idempotency() {
	input := s.adrExample()

	first, err := s.execute(input)
	require.NoError(s.T(), err)

	input.DestinationAmount = decimal.NewFromInt(91)
	second, err := s.execute(input)
	require.NoError(s.T(), err)

	require.Equal(s.T(), first.JournalEntryID, second.JournalEntryID)
	require.Equal(s.T(), 1, s.Count[model.JournalEntry]())
	require.Len(s.T(), s.postingsOf(first.JournalEntryID), 5)
	require.Contains(s.T(), s.postingsOf(first.JournalEntryID), fmt.Sprintf("debit account %d 90", s.revolutEUR.ID))
}

func (s *UnitTestSuite) Test_RegisterFXConversion_KeyOfAnotherKindOfEntry() {
	input := s.adrExample()
	temporaltesting.Insert(s.T(), s.DB, &model.JournalEntry{
		IdempotencyKey: ptr.NewPtr(input.IdempotencyKey),
		OccurredAt:     input.OccurredAt,
		Status:         model.JournalEntryDraft,
	})

	_, err := s.execute(input)

	require.ErrorContains(s.T(), err, "not an FX conversion")
}

func (s *UnitTestSuite) Test_RegisterFXConversion_Invalid() {
	tests := map[string]struct {
		mutate func(*lt.RegisterFXConversionInput)
		expMsg string
	}{
		"missing idempotency key": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.IdempotencyKey = "" },
			expMsg: "IdempotencyKey",
		},
		"missing occurrence date": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.OccurredAt = time.Time{} },
			expMsg: "OccurredAt",
		},
		"zero source amount": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.SourceAmount = decimal.Zero },
			expMsg: "must be positive",
		},
		"negative destination amount": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.DestinationAmount = decimal.NewFromInt(-90) },
			expMsg: "must be positive",
		},
		"zero fee": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.Fee.Amount = decimal.Zero },
			expMsg: "fee amount must be positive",
		},
		"fee equal to the source amount": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.Fee.Amount = i.SourceAmount },
			expMsg: "fee must be smaller",
		},
		"clearing account reused as source": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.SourceClearingLedgerAccountID = i.SourceLedgerAccountID },
			expMsg: "must all be different",
		},
		"non positive account id": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.SourceLedgerAccountID = 0 },
			expMsg: "must be positive",
		},
		"source and destination share a currency": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.DestinationLedgerAccountID = s.savingsUSD.ID },
			expMsg: "different currencies",
		},
		"unknown account": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.SourceLedgerAccountID = 999999 },
			expMsg: "not found",
		},
		"closed source account": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.SourceLedgerAccountID = s.closedUSD.ID },
			expMsg: "is closed",
		},
		"source is not an asset account": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.SourceLedgerAccountID = s.usdIncome.ID },
			expMsg: "source ledger account must be of kind asset",
		},
		"source clearing in the wrong currency": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.SourceClearingLedgerAccountID = s.savingsEUR.ID },
			expMsg: "source clearing ledger account must be denominated",
		},
		"fee account is not an expense": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.Fee.LedgerAccountID = s.savingsUSD.ID },
			expMsg: "fee ledger account must be of kind expense",
		},
		"fee account in the destination currency": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.Fee.LedgerAccountID = s.eurFees.ID },
			expMsg: "fee ledger account must be denominated",
		},
		"amount beyond the currency precision": {
			mutate: func(i *lt.RegisterFXConversionInput) { i.DestinationAmount = decimal.RequireFromString("90.001") },
			expMsg: "exceeds the 2 decimal places",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			input := s.adrExample()
			tc.mutate(&input)

			_, err := s.execute(input)

			require.ErrorContains(s.T(), err, tc.expMsg)
			require.Zero(s.T(), s.Count[model.JournalEntry](), "a rejected conversion must leave nothing behind")
		})
	}
}
