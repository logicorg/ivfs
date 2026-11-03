// Package scenarios drives the client-facing workflows through stories that
// span several commands, against the real activities and database, and reads
// the result back through the supported queries.
package scenarios

import (
	"testing"
	"time"

	ledgerjournal "github.com/logicorg/ivfs/finance/activities/ledger/journal"
	ledgerportfolio "github.com/logicorg/ivfs/finance/activities/ledger/portfolio"
	ledgerreporting "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	portfoliowf "github.com/logicorg/ivfs/finance/workflows/ledger/portfolio"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var vestDay = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite

	portfolio *ledgerportfolio.Activities
	journal   *ledgerjournal.Activities
	reporting *ledgerreporting.Activities

	eur              *model.Currency
	financialAccount *model.FinancialAccount
	asset            *model.Asset

	// holding carries the value of the units held and brokerCash the cash in the
	// broker account, and both belong to the financial account. bankCash is where
	// payroll pays from.
	holding, brokerCash, bankCash *model.LedgerAccount
	// rsuIncome and esppIncome are where the compensation of each plan is
	// booked, and tax is where the tax withheld from it is.
	rsuIncome, esppIncome, tax *model.LedgerAccount
}

func (s *UnitTestSuite) SetupTest() {
	s.portfolio = &ledgerportfolio.Activities{Pools: tenancy.Static(s.DB)}
	s.journal = &ledgerjournal.Activities{Pools: tenancy.Static(s.DB)}
	s.reporting = &ledgerreporting.Activities{Pools: tenancy.Static(s.DB)}
}

func (s *UnitTestSuite) BeforeTest(_, _ string) {
	sqllib.InitDatabaseT(s.T(), s.DB)

	s.eur = testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, s.eur)

	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	s.financialAccount = &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	temporaltesting.Insert(s.T(), s.DB, s.financialAccount)
	s.asset = &model.Asset{Name: "ACME", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, s.asset)

	account := func(name string, kind model.LedgerAccountKind, financialAccount *model.FinancialAccount) *model.LedgerAccount {
		ledgerAccount := &model.LedgerAccount{CurrencyID: s.eur.ID, Name: name, Kind: kind}
		if financialAccount != nil {
			ledgerAccount.FinancialAccountID = &financialAccount.ID
		}
		temporaltesting.Insert(s.T(), s.DB, ledgerAccount)
		return ledgerAccount
	}
	s.holding = account("investments", model.LedgerAccountAsset, s.financialAccount)
	s.brokerCash = account("broker cash", model.LedgerAccountAsset, s.financialAccount)
	s.bankCash = account("bank cash", model.LedgerAccountAsset, nil)
	s.rsuIncome = account("RSU income", model.LedgerAccountIncome, nil)
	s.esppIncome = account("ESPP discount", model.LedgerAccountIncome, nil)
	s.tax = account("tax withheld", model.LedgerAccountExpense, nil)
}

func (s *UnitTestSuite) AfterTest(_, _ string) { s.DropAllTables() }

func Test_UnitTestSuite(t *testing.T) { suite.Run(t, new(UnitTestSuite)) }

// execute runs wf on a fresh environment wired to the real activities rather
// than mocks, so each call exercises the whole path down to the database. result
// may be nil for workflows that only return an error.
func (s *UnitTestSuite) execute(wf, input, result any) error {
	s.T().Helper()

	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(s.portfolio)
	env.RegisterActivity(s.journal)
	env.RegisterActivity(s.reporting)

	env.ExecuteWorkflow(wf, input)
	if err := env.GetWorkflowError(); err != nil {
		return err
	}
	if result != nil {
		require.NoError(s.T(), env.GetWorkflowResult(result))
	}
	return nil
}

func (s *UnitTestSuite) runWorkflow(wf, input, result any) {
	s.T().Helper()

	require.NoError(s.T(), s.execute(wf, input, result))
}

func debit(account *model.LedgerAccount, amount int64) lt.PostingInput {
	return lt.PostingInput{LedgerAccountID: account.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingDebit}
}

func credit(account *model.LedgerAccount, amount int64) lt.PostingInput {
	return lt.PostingInput{LedgerAccountID: account.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingCredit}
}

// vest is a vest of 100 units worth 10 each, all delivered to the account, that
// books their 1000 as income.
func (s *UnitTestSuite) vest(key string) lt.RegisterVestInput {
	return lt.RegisterVestInput{
		FinancialAccountID:  s.financialAccount.ID,
		AssetID:             s.asset.ID,
		Units:               decimal.NewFromInt(100),
		CostBasisAmount:     decimal.NewFromInt(1000),
		CostBasisCurrencyID: s.eur.ID,
		Entry: lt.PostJournalEntryInput{
			IdempotencyKey: key,
			OccurredAt:     vestDay,
			Postings:       []lt.PostingInput{credit(s.rsuIncome, 1000), debit(s.holding, 1000)},
		},
	}
}

// sellToCover sells 50 of the units at 10 each, an hour after the vest. 400 of
// the 500 proceeds pay the tax and 100 reach the broker cash account, and the
// 500 of cost basis removed is credited to the holding.
func (s *UnitTestSuite) sellToCover(key string) lt.RegisterInvestmentTradeInput {
	return lt.RegisterInvestmentTradeInput{
		FinancialAccountID:  s.financialAccount.ID,
		AssetID:             s.asset.ID,
		UnitsDelta:          decimal.NewFromInt(-50),
		CostBasisAmount:     ptr.NewPtr(decimal.NewFromInt(-500)),
		CostBasisCurrencyID: ptr.Int64(s.eur.ID),
		Entry: lt.PostJournalEntryInput{
			IdempotencyKey: key,
			OccurredAt:     vestDay.Add(time.Hour),
			Postings:       []lt.PostingInput{debit(s.tax, 400), debit(s.brokerCash, 100), credit(s.holding, 500)},
		},
	}
}

func (s *UnitTestSuite) balance(account *model.LedgerAccount) decimal.Decimal {
	s.T().Helper()

	var result lt.LedgerAccountBalanceResult
	s.runWorkflow(reportingwf.LedgerAccountBalance, lt.LedgerAccountBalanceParams{LedgerAccountID: account.ID}, &result)
	return result.Balance
}

func (s *UnitTestSuite) unitsHeld() decimal.Decimal {
	s.T().Helper()

	var result lt.PositionAtResult
	s.runWorkflow(portfoliowf.PositionAt, lt.PositionAtParams{FinancialAccountID: s.financialAccount.ID, AssetID: s.asset.ID}, &result)
	return result.Units
}

func (s *UnitTestSuite) history() []model.PositionMovement {
	s.T().Helper()

	var result lt.ListPositionMovementsResult
	s.runWorkflow(portfoliowf.ListPositionMovements, lt.ListPositionMovementsParams{FinancialAccountID: s.financialAccount.ID, AssetID: s.asset.ID}, &result)
	return result.Movements
}

// costBasis is the cost basis of the units held, which is the sum of the basis
// of every movement in the history. It also checks that they are all in euros,
// since a sum across currencies would mean nothing.
func (s *UnitTestSuite) costBasis() decimal.Decimal {
	s.T().Helper()

	total := decimal.Zero
	for _, movement := range s.history() {
		require.NotNil(s.T(), movement.CostBasisAmount)
		require.Equal(s.T(), s.eur.ID, *movement.CostBasisCurrencyID)
		total = total.Add(*movement.CostBasisAmount)
	}
	return total
}

func requireAmount(t *testing.T, want int64, got decimal.Decimal, what string) {
	t.Helper()

	require.True(t, decimal.NewFromInt(want).Equal(got), "%s is %s, want %d", what, got, want)
}

// Test_VestWithSellToCover records a vest whose tax is paid by selling part of
// the units at once, as two commands, and checks that the units held, the
// compensation income, the tax withheld, the cash proceeds, and the cost basis
// can all be read back from the ledger.
func (s *UnitTestSuite) Test_VestWithSellToCover() {
	s.runWorkflow(portfoliowf.RegisterVest, s.vest("acme-rsu-2026-03-01"), nil)
	s.runWorkflow(portfoliowf.RegisterInvestmentTrade, s.sellToCover("acme-rsu-2026-03-01-cover"), nil)

	requireAmount(s.T(), 50, s.unitsHeld(), "units held")
	requireAmount(s.T(), 1000, s.balance(s.rsuIncome), "compensation income")
	requireAmount(s.T(), 400, s.balance(s.tax), "tax withheld")
	requireAmount(s.T(), 100, s.balance(s.brokerCash), "cash proceeds")
	requireAmount(s.T(), 500, s.balance(s.holding), "value of the units held")
	requireAmount(s.T(), 500, s.costBasis(), "cost basis")

	history := s.history()
	require.Len(s.T(), history, 2)
	require.Equal(s.T(), model.PositionMovementVest, history[0].Kind)
	require.Equal(s.T(), model.PositionMovementTrade, history[1].Kind)
}

// Test_EsppPurchase records an ESPP purchase as an ordinary trade: 10 units worth
// 20 each are bought for 17 each through payroll, and the discount is income. It
// checks that the units, the cost basis, the discount, and the cash paid can all
// be read back.
func (s *UnitTestSuite) Test_EsppPurchase() {
	purchase := lt.RegisterInvestmentTradeInput{
		FinancialAccountID:  s.financialAccount.ID,
		AssetID:             s.asset.ID,
		UnitsDelta:          decimal.NewFromInt(10),
		CostBasisAmount:     ptr.NewPtr(decimal.NewFromInt(200)),
		CostBasisCurrencyID: ptr.Int64(s.eur.ID),
		Entry: lt.PostJournalEntryInput{
			IdempotencyKey: "acme-espp-2026-03-01",
			OccurredAt:     vestDay,
			Postings:       []lt.PostingInput{debit(s.holding, 200), credit(s.bankCash, 170), credit(s.esppIncome, 30)},
		},
	}

	s.runWorkflow(portfoliowf.RegisterInvestmentTrade, purchase, nil)

	requireAmount(s.T(), 10, s.unitsHeld(), "units held")
	requireAmount(s.T(), 200, s.costBasis(), "cost basis")
	requireAmount(s.T(), 30, s.balance(s.esppIncome), "discount income")
	requireAmount(s.T(), -170, s.balance(s.bankCash), "cash paid")
}

// Test_CorrectAVest checks that a vest is corrected by reversing its entry.
func (s *UnitTestSuite) Test_CorrectAVest() {
	var vest lt.RegisterVestResult
	s.runWorkflow(portfoliowf.RegisterVest, s.vest("acme-rsu-2026-03-01"), &vest)
	var sale lt.RegisterInvestmentTradeResult
	s.runWorkflow(portfoliowf.RegisterInvestmentTrade, s.sellToCover("acme-rsu-2026-03-01-cover"), &sale)

	res := new(lt.ReverseJournalEntryResult)
	err := s.execute(journalwf.ReverseJournalEntry, lt.ReverseJournalEntryInput{JournalEntryID: vest.JournalEntryID, IdempotencyKey: "undo-vest"}, res)

	require.NoError(s.T(), err)
	// reversal of vest still leaves the sale, so the units held are negative and the compensation income is gone.
	requireAmount(s.T(), -50, s.unitsHeld(), "units held after the reversal")
	requireAmount(s.T(), 0, s.balance(s.rsuIncome), "compensation income after the reversal")
	require.Len(s.T(), res.Warnings, 1, "expecting a warning about negative holdings")

	s.runWorkflow(journalwf.ReverseJournalEntry, lt.ReverseJournalEntryInput{JournalEntryID: sale.JournalEntryID, IdempotencyKey: "undo-cover"}, nil)

	requireAmount(s.T(), 0, s.unitsHeld(), "units held")
	requireAmount(s.T(), 0, s.balance(s.rsuIncome), "compensation income")
	requireAmount(s.T(), 0, s.balance(s.tax), "tax withheld")
	requireAmount(s.T(), 0, s.balance(s.brokerCash), "cash proceeds")
	requireAmount(s.T(), 0, s.costBasis(), "cost basis")
}
