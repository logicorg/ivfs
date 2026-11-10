package journal

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) TestPostJournalEntry() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	debitAccount := testLedgerAccount(s.T(), s.DB, eur.ID, "cash", model.LedgerAccountAsset)
	creditAccount := testLedgerAccount(s.T(), s.DB, eur.ID, "expense", model.LedgerAccountExpense)

	input := lt.PostJournalEntryInput{
		IdempotencyKey: "post-groceries-1",
		OccurredAt:     time.Now().UTC(),
		Description:    ptr.NewPtr("groceries"),
		Metadata:       map[string]string{"source": "test"},
		Postings: []lt.PostingInput{
			{LedgerAccountID: debitAccount.ID, Amount: decimal.NewFromInt(45), Side: model.PostingDebit},
			{LedgerAccountID: creditAccount.ID, Amount: decimal.NewFromInt(45), Side: model.PostingCredit},
		},
	}

	activityResult, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, input)
	require.NoError(s.T(), err)
	var result lt.PostJournalEntryResult
	require.NoError(s.T(), activityResult.Get(&result))
	require.NotZero(s.T(), result.JournalEntryID)

	var entry model.JournalEntry
	require.NoError(s.T(), s.DB.NewSelect().Model(&entry).Where("id = ?", result.JournalEntryID).Scan(s.T().Context()))
	require.Equal(s.T(), model.JournalEntryPosted, entry.Status)
	require.NotNil(s.T(), entry.PostedAt)

	var postings []model.Posting
	require.NoError(s.T(), s.DB.NewSelect().Model(&postings).Where("journal_entry_id = ?", result.JournalEntryID).Order("line_number").Scan(s.T().Context()))
	require.Len(s.T(), postings, 2)
	require.Equal(s.T(), 1, postings[0].LineNumber)
	require.True(s.T(), postings[0].Amount.Equal(decimal.NewFromInt(45)))
}

func (s *UnitTestSuite) TestPostJournalEntry_AmountPrecision() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	dinar := testhelpers.BHD()
	temporaltesting.Insert(s.T(), s.DB, dinar)
	yen := testhelpers.JapaneseYen()
	temporaltesting.Insert(s.T(), s.DB, yen)

	tests := map[string]struct {
		currency *model.Currency
		amount   string
		expErr   bool
	}{
		"whole amount":                      {currency: euro, amount: "45"},
		"amount at the currency precision":  {currency: euro, amount: "45.67"},
		"trailing zeros beyond precision":   {currency: euro, amount: "45.670"},
		"one decimal beyond precision":      {currency: euro, amount: "45.675", expErr: true},
		"smallest fraction beyond a cent":   {currency: euro, amount: "0.001", expErr: true},
		"three decimals at dinar precision": {currency: dinar, amount: "45.675"},
		"four decimals beyond dinar":        {currency: dinar, amount: "45.6755", expErr: true},
		"whole amount on a yen":             {currency: yen, amount: "100"},
		"any decimal on a yen":              {currency: yen, amount: "100.5", expErr: true},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			amount := decimal.RequireFromString(tc.amount)
			debitAccount := testLedgerAccount(s.T(), s.DB, tc.currency.ID, "debit "+name, model.LedgerAccountAsset)
			creditAccount := testLedgerAccount(s.T(), s.DB, tc.currency.ID, "credit "+name, model.LedgerAccountExpense)
			input := lt.PostJournalEntryInput{IdempotencyKey: name, OccurredAt: time.Now().UTC(), Postings: []lt.PostingInput{
				{LedgerAccountID: debitAccount.ID, Amount: amount, Side: model.PostingDebit},
				{LedgerAccountID: creditAccount.ID, Amount: amount, Side: model.PostingCredit},
			}}

			_, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, input)

			count := s.CountWhere[model.JournalEntry]("idempotency_key = ?", name)

			if tc.expErr {
				require.Error(s.T(), err)
				require.Zero(s.T(), count, "a rejected entry must leave nothing behind")
			} else {
				require.NoError(s.T(), err)
				require.EqualValues(s.T(), 1, count)
			}
		})
	}
}

func (s *UnitTestSuite) TestPostJournalEntry_ClosedLedgerAccount() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	open := testLedgerAccount(s.T(), s.DB, euro.ID, "cash", model.LedgerAccountAsset)
	closedAt := time.Now().UTC()
	closed := &model.LedgerAccount{CurrencyID: euro.ID, Name: "closed", Kind: model.LedgerAccountExpense, ClosedAt: &closedAt}
	temporaltesting.Insert(s.T(), s.DB, closed)

	_, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{IdempotencyKey: "into-closed", OccurredAt: time.Now().UTC(), Postings: []lt.PostingInput{
		{LedgerAccountID: open.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
		{LedgerAccountID: closed.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
	}})

	require.ErrorContains(s.T(), err, "is closed")
	require.Zero(s.T(), s.Count[model.JournalEntry](), "a rejected entry must leave nothing behind")
}

// A posting that starts while a transaction is closing one of its accounts has
// to wait for it and then see the account closed, rather than read the stale
// open row and post into a closed account.
func (s *UnitTestSuite) TestPostJournalEntry_WaitsForConcurrentClose() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	debit := testLedgerAccount(s.T(), s.DB, euro.ID, "cash", model.LedgerAccountAsset)
	credit := testLedgerAccount(s.T(), s.DB, euro.ID, "expense", model.LedgerAccountExpense)
	journalEntryInput := lt.PostJournalEntryInput{
		IdempotencyKey: "racing-post",
		OccurredAt:     time.Now().UTC(),
		Postings: []lt.PostingInput{
			{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
			{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
		},
	}

	closing, err := s.DB.BeginTx(s.T().Context(), nil)
	require.NoError(s.T(), err)
	defer func() { _ = closing.Rollback() }()

	_, err = closing.ExecContext(s.T().Context(), "UPDATE ledger.ledger_account SET closed_at = now() WHERE id = ?", credit.ID)
	require.NoError(s.T(), err)

	posted := make(chan error, 1)
	go func() {
		_, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, journalEntryInput)
		posted <- err
	}()

	select {
	case <-posted:
		s.T().Fatal("the posting finished without waiting for the transaction closing its account")
	case <-time.After(500 * time.Millisecond):
	}

	require.NoError(s.T(), closing.Commit())
	require.ErrorContains(s.T(), <-posted, "is closed")
}

func (s *UnitTestSuite) TestPostJournalEntry_Idempotency() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	debitAccount := testLedgerAccount(s.T(), s.DB, euro.ID, "cash", model.LedgerAccountAsset)
	creditAccount := testLedgerAccount(s.T(), s.DB, euro.ID, "offset", model.LedgerAccountExpense)
	input := lt.PostJournalEntryInput{IdempotencyKey: "same-entry", OccurredAt: time.Now().UTC(), Postings: []lt.PostingInput{
		{LedgerAccountID: debitAccount.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
		{LedgerAccountID: creditAccount.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
	}}
	first, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, input)
	require.NoError(s.T(), err)
	var firstResult lt.PostJournalEntryResult
	require.NoError(s.T(), first.Get(&firstResult))
	second, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, input)
	require.NoError(s.T(), err)
	var secondResult lt.PostJournalEntryResult
	require.NoError(s.T(), second.Get(&secondResult))
	require.Equal(s.T(), firstResult.JournalEntryID, secondResult.JournalEntryID)

	require.Equal(s.T(), 1, s.CountWhere[model.JournalEntry]("idempotency_key = ?", input.IdempotencyKey))
}

func (s *UnitTestSuite) TestPostJournalEntry_CurrencyBalancing() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	dollar := testhelpers.USDollar()
	temporaltesting.Insert(s.T(), s.DB, dollar)

	eurDebit := testLedgerAccount(s.T(), s.DB, euro.ID, "eur debit", model.LedgerAccountAsset)
	eurCredit := testLedgerAccount(s.T(), s.DB, euro.ID, "eur credit", model.LedgerAccountExpense)

	usdDebit := testLedgerAccount(s.T(), s.DB, dollar.ID, "usd debit", model.LedgerAccountAsset)
	usdCredit := testLedgerAccount(s.T(), s.DB, dollar.ID, "usd credit", model.LedgerAccountExpense)

	tests := map[string]struct {
		postings []lt.PostingInput
		wantErr  bool
	}{
		"balanced per currency": {
			postings: []lt.PostingInput{
				{LedgerAccountID: eurDebit.ID, Amount: decimal.NewFromInt(90), Side: model.PostingDebit},
				{LedgerAccountID: eurCredit.ID, Amount: decimal.NewFromInt(90), Side: model.PostingCredit},
				{LedgerAccountID: usdDebit.ID, Amount: decimal.NewFromInt(100), Side: model.PostingDebit},
				{LedgerAccountID: usdCredit.ID, Amount: decimal.NewFromInt(100), Side: model.PostingCredit},
			},
		},
		"unbalanced per currency": {
			postings: []lt.PostingInput{
				{LedgerAccountID: eurDebit.ID, Amount: decimal.NewFromInt(90), Side: model.PostingDebit},
				{LedgerAccountID: eurCredit.ID, Amount: decimal.NewFromInt(89), Side: model.PostingCredit},
			},
			wantErr: true,
		},
	}
	for name, test := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{IdempotencyKey: "currency-" + name, OccurredAt: time.Now().UTC(), Postings: test.postings})
			if test.wantErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
			}
		})
	}
}
func (s *UnitTestSuite) TestPostJournalEntryValidation() {
	tests := []struct {
		name string
		in   lt.PostJournalEntryInput
	}{
		{
			name: "missing occurrence date",
			in:   lt.PostJournalEntryInput{Postings: make([]lt.PostingInput, 2)},
		},
		{
			name: "fewer than two postings",
			in: lt.PostJournalEntryInput{
				OccurredAt: time.Now(),
				Postings:   make([]lt.PostingInput, 1),
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, tt.in)
			require.Error(s.T(), err)
		})
	}
}

// Closing an account is final, so new entries into it are rejected.
func (s *UnitTestSuite) TestPostJournalEntry_AfterAccountIsClosed() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	debit := testLedgerAccount(s.T(), s.DB, euro.ID, "asset", model.LedgerAccountAsset)
	credit := testLedgerAccount(s.T(), s.DB, euro.ID, "income", model.LedgerAccountIncome)
	postings := []lt.PostingInput{
		{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(25), Side: model.PostingDebit},
		{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(25), Side: model.PostingCredit},
	}

	_, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{IdempotencyKey: "before-close", OccurredAt: time.Now().UTC(), Postings: postings})
	require.NoError(s.T(), err)

	closedAt := time.Now().UTC()
	credit.ClosedAt = &closedAt
	_, err = s.DB.NewUpdate().Model(credit).Column("closed_at").WherePK().Exec(s.T().Context())
	require.NoError(s.T(), err)

	_, err = s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{IdempotencyKey: "after-close", OccurredAt: time.Now().UTC(), Postings: postings})
	require.ErrorContains(s.T(), err, "is closed")
}

func (s *UnitTestSuite) TestPostJournalEntry_UnknownCounterparty() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	debit := testLedgerAccount(s.T(), s.DB, euro.ID, "cash", model.LedgerAccountAsset)
	credit := testLedgerAccount(s.T(), s.DB, euro.ID, "expense", model.LedgerAccountExpense)

	_, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{
		IdempotencyKey: "unknown-counterparty",
		OccurredAt:     time.Now().UTC(),
		CounterpartyID: ptr.Int64(999999),
		Postings: []lt.PostingInput{
			{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
			{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
		},
	})

	var appErr *temporal.ApplicationError
	require.ErrorAs(s.T(), err, &appErr)
	require.Equal(s.T(), "ErrNotFound", appErr.Type())
	require.ErrorContains(s.T(), err, "Counterparty")
}
