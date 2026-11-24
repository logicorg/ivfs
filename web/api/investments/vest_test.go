package investments

import (
	"net/http"
	"testing"
	"time"

	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	investmentswf "github.com/logicorg/ivfs/finance/workflows/facade/investments"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestVestAsset checks that the form is passed to RecordInvestment as a vest,
// with its decimals exact (10.00 arrives as 10) and its withholding, tax
// category and note only when given, that the entry and the movement are
// returned, that a refusal keeps its status, and that a form Huma rejects runs
// no workflow.
func TestVestAsset(t *testing.T) {
	const form = `{"idempotencyKey":"a71f","financialAccountId":20,"assetId":7,"unitsVested":"100","fairMarketValue":"10.00",` +
		`"currencyCode":"EUR","incomeCategoryId":600,"occurredAt":"2026-10-03T10:00:00Z"`
	rsu := it.VestInput{
		IdempotencyKey: "a71f", FinancialAccountID: 20, AssetID: 7, UnitsVested: decimal.NewFromInt(100),
		FairMarketValue: decimal.NewFromInt(10), CurrencyCode: "EUR", IncomeCategoryID: 600,
		OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
	}
	withTags := rsu
	withTags.TagIDs = []int64{3, 4}
	withheld := rsu
	withheld.UnitsWithheld, withheld.TaxCategoryID = decimal.NewFromInt(40), 601
	withheld.Description = "Q3 vest"
	vest := func(params it.VestInput, result it.TradeResult, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{Workflow: investmentswf.RecordInvestment,
			Params: it.RecordInvestmentInput{Kind: it.KindVest, Vest: &params}, Result: result, Err: err}
	}
	recorded := it.TradeResult{JournalEntryID: 61, PositionMovementID: 62}
	const wrongKind = `"Tax withheld" is an expense category, not an income one`

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"record a vest": {
			Method: http.MethodPost, Path: "/api/investments/vests", Body: form + `}`,
			Call:       vest(rsu, recorded, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":61,"positionMovementId":62}`,
		},
		"with tags": {
			Method: http.MethodPost, Path: "/api/investments/vests", Body: form + `,"tagIds":[3,4]}`,
			Call:       vest(withTags, recorded, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":61,"positionMovementId":62}`,
		},
		"with units withheld for tax and a note": {
			Method: http.MethodPost, Path: "/api/investments/vests",
			Body:       form + `,"unitsWithheld":"40","taxCategoryId":601,"description":"Q3 vest"}`,
			Call:       vest(withheld, recorded, nil),
			WantStatus: http.StatusOK, WantBody: `{"journalEntryId":61,"positionMovementId":62}`,
		},
		"a category of the other kind": {
			Method: http.MethodPost, Path: "/api/investments/vests", Body: form + `}`,
			Call:       vest(rsu, it.TradeResult{}, temporal.NewApplicationError(wrongKind, "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, wrongKind),
		},
		"an unknown category": {
			Method: http.MethodPost, Path: "/api/investments/vests", Body: form + `}`,
			Call:       vest(rsu, it.TradeResult{}, apitest.ChildWorkflowError("ErrNotFound", "resource Category not found with ID: 600")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Category not found with ID: 600"),
		},
		"a closed account": {
			Method: http.MethodPost, Path: "/api/investments/vests", Body: form + `}`,
			Call:       vest(rsu, it.TradeResult{}, apitest.ChildWorkflowError("ErrResourceClosed", "resource FinancialAccount with ID: 20 is closed")),
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource FinancialAccount with ID: 20 is closed"),
		},
		"no income category": {
			Method: http.MethodPost, Path: "/api/investments/vests",
			Body:       `{"idempotencyKey":"a71f","financialAccountId":20,"assetId":7,"unitsVested":"100","fairMarketValue":"10.00","currencyCode":"EUR","occurredAt":"2026-10-03T10:00:00Z"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"units withheld that are not a decimal": {
			Method: http.MethodPost, Path: "/api/investments/vests", Body: form + `,"unitsWithheld":"forty"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"a tax category of zero": {
			Method: http.MethodPost, Path: "/api/investments/vests", Body: form + `,"taxCategoryId":0}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
