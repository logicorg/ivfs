package investments

import (
	"net/http"
	"testing"
	"time"

	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	investmentswf "github.com/logicorg/ivfs/finance/workflows/facade/investments"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestBuyAsset checks that the form is passed to RecordInvestment as a buy,
// with its decimals exact (3.00 arrives as 3) and its fee and note only when
// given, that the entry, the movement, and any warning are returned, that a
// refusal keeps its status, and that a form Huma rejects runs no workflow.
func TestBuyAsset(t *testing.T) {
	const form = `{"idempotencyKey":"5f2a","financialAccountId":20,"assetId":7,"units":"10","amount":"1050.00",` +
		`"currencyCode":"EUR","occurredAt":"2026-10-03T10:00:00Z"`
	vwce := it.BuyAssetInput{
		IdempotencyKey: "5f2a", FinancialAccountID: 20, AssetID: 7, Units: decimal.NewFromInt(10),
		Amount: decimal.RequireFromString("1050"), CurrencyCode: "EUR", OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
	}
	withFeeAndNote := vwce
	withFeeAndNote.Fee = decimal.RequireFromString("3")
	withFeeAndNote.Description = "monthly plan"
	buy := func(params it.BuyAssetInput, result it.TradeResult, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{Workflow: investmentswf.RecordInvestment,
			Params: it.RecordInvestmentInput{Kind: it.KindBuy, Buy: &params}, Result: result, Err: err}
	}
	recorded := it.TradeResult{JournalEntryID: 41, PositionMovementID: 42}
	const missingBalances = `financial account 20 has no "Cash" and "Investments" balances in EUR: ` +
		`open it again as a brokerage with EUR to add them`

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"record a buy": {
			Method: http.MethodPost, Path: "/api/investments/buys", Body: form + `}`,
			Call:       buy(vwce, recorded, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":41,"positionMovementId":42}`,
		},
		"with a fee and a note": {
			Method: http.MethodPost, Path: "/api/investments/buys", Body: form + `,"fee":"3.00","description":"monthly plan"}`,
			Call:       buy(withFeeAndNote, recorded, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":41,"positionMovementId":42}`,
		},
		"a buy that leaves a holding negative": {
			Method: http.MethodPost, Path: "/api/investments/buys", Body: form + `}`,
			Call: buy(vwce, it.TradeResult{JournalEntryID: 41, PositionMovementID: 42, Warnings: []lt.NegativeHolding{{
				FinancialAccountID: 20, FinancialAccountName: "IBKR", AssetID: 7, AssetName: "VWCE",
				At: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), Units: decimal.NewFromInt(-2),
			}}}, nil),
			WantStatus: http.StatusOK,
			WantBody: `{"journalEntryId":41,"positionMovementId":42,"warnings":[{"financialAccountId":20,` +
				`"financialAccountName":"IBKR","assetId":7,"assetName":"VWCE","at":"2026-11-01T00:00:00Z","units":"-2"}]}`,
		},
		"an account without the balances": {
			Method: http.MethodPost, Path: "/api/investments/buys", Body: form + `}`,
			Call:       buy(vwce, it.TradeResult{}, temporal.NewApplicationError(missingBalances, "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, missingBalances),
		},
		"an unknown asset": {
			Method: http.MethodPost, Path: "/api/investments/buys", Body: form + `}`,
			Call:       buy(vwce, it.TradeResult{}, apitest.ChildWorkflowError("ErrNotFound", "resource Asset not found with ID: 7")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Asset not found with ID: 7"),
		},
		"a closed balance": {
			Method: http.MethodPost, Path: "/api/investments/buys", Body: form + `}`,
			Call:       buy(vwce, it.TradeResult{}, apitest.ChildWorkflowError("ErrResourceClosed", "resource LedgerAccount is closed")),
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource LedgerAccount is closed"),
		},
		"no financial account": {
			Method: http.MethodPost, Path: "/api/investments/buys",
			Body:       `{"idempotencyKey":"5f2a","financialAccountId":0,"assetId":7,"units":"10","amount":"1050.00","currencyCode":"EUR","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"units that are not a decimal": {
			Method: http.MethodPost, Path: "/api/investments/buys",
			Body:       `{"idempotencyKey":"5f2a","financialAccountId":20,"assetId":7,"units":"ten","amount":"1050.00","currencyCode":"EUR","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"no currency": {
			Method: http.MethodPost, Path: "/api/investments/buys",
			Body:       `{"idempotencyKey":"5f2a","financialAccountId":20,"assetId":7,"units":"10","amount":"1050.00","currencyCode":"","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
