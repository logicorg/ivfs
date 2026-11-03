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

// TestSellAsset checks that the form is passed to RecordInvestment as a sale,
// with its decimals exact (0.50 arrives as 0.5) and its fee and note only when
// given, that the entry, the movement, and the warning of an oversold holding
// are returned, that a refusal keeps its status, and that a form Huma rejects
// runs no workflow.
func TestSellAsset(t *testing.T) {
	const form = `{"idempotencyKey":"8e41","financialAccountId":20,"assetId":7,"units":"0.50","amount":"60.25",` +
		`"currencyCode":"EUR","occurredAt":"2026-10-03T10:00:00Z"`
	vwce := it.SellAssetInput{
		IdempotencyKey: "8e41", FinancialAccountID: 20, AssetID: 7, Units: decimal.RequireFromString("0.5"),
		Amount: decimal.RequireFromString("60.25"), CurrencyCode: "EUR", OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
	}
	withFeeAndNote := vwce
	withFeeAndNote.Fee = decimal.RequireFromString("1.5")
	withFeeAndNote.Description = "sell to cover"
	sell := func(params it.SellAssetInput, result it.TradeResult, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{Workflow: investmentswf.RecordInvestment,
			Params: it.RecordInvestmentInput{Kind: it.KindSell, Sell: &params}, Result: result, Err: err}
	}
	recorded := it.TradeResult{JournalEntryID: 51, PositionMovementID: 52}
	const feeTooHigh = "the fee of a sale must be less than its amount, so the sale leaves some proceeds"

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"record a sale": {
			Method: http.MethodPost, Path: "/api/investments/sells", Body: form + `}`,
			Call:       sell(vwce, recorded, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":51,"positionMovementId":52}`,
		},
		"with a fee and a note": {
			Method: http.MethodPost, Path: "/api/investments/sells", Body: form + `,"fee":"1.50","description":"sell to cover"}`,
			Call:       sell(withFeeAndNote, recorded, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":51,"positionMovementId":52}`,
		},
		"a sale of more units than are held": {
			Method: http.MethodPost, Path: "/api/investments/sells", Body: form + `}`,
			Call: sell(vwce, it.TradeResult{JournalEntryID: 51, PositionMovementID: 52, Warnings: []lt.NegativeHolding{{
				FinancialAccountID: 20, FinancialAccountName: "IBKR", AssetID: 7, AssetName: "VWCE",
				At: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC), Units: decimal.RequireFromString("-0.25"),
			}}}, nil),
			WantStatus: http.StatusOK,
			WantBody: `{"journalEntryId":51,"positionMovementId":52,"warnings":[{"financialAccountId":20,` +
				`"financialAccountName":"IBKR","assetId":7,"assetName":"VWCE","at":"2026-10-03T10:00:00Z","units":"-0.25"}]}`,
		},
		"a fee that takes all the proceeds": {
			Method: http.MethodPost, Path: "/api/investments/sells", Body: form + `}`,
			Call:       sell(vwce, it.TradeResult{}, temporal.NewApplicationError(feeTooHigh, "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, feeTooHigh),
		},
		"an unknown currency": {
			Method: http.MethodPost, Path: "/api/investments/sells", Body: form + `}`,
			Call:       sell(vwce, it.TradeResult{}, apitest.ChildWorkflowError("ErrNotFound", "resource Currency not found with code: EUR")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Currency not found with code: EUR"),
		},
		"a closed balance": {
			Method: http.MethodPost, Path: "/api/investments/sells", Body: form + `}`,
			Call:       sell(vwce, it.TradeResult{}, apitest.ChildWorkflowError("ErrResourceClosed", "resource LedgerAccount is closed")),
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource LedgerAccount is closed"),
		},
		"no asset": {
			Method: http.MethodPost, Path: "/api/investments/sells",
			Body:       `{"idempotencyKey":"8e41","financialAccountId":20,"assetId":0,"units":"0.50","amount":"60.25","currencyCode":"EUR","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"an amount that is not a decimal": {
			Method: http.MethodPost, Path: "/api/investments/sells",
			Body:       `{"idempotencyKey":"8e41","financialAccountId":20,"assetId":7,"units":"0.50","amount":"60,25","currencyCode":"EUR","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"no occurrence date": {
			Method: http.MethodPost, Path: "/api/investments/sells",
			Body:       `{"idempotencyKey":"8e41","financialAccountId":20,"assetId":7,"units":"0.50","amount":"60.25","currencyCode":"EUR"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
