package currencies

import (
	"time"

	"github.com/logicorg/ivfs/finance/activities/currency/conversion"
	ct "github.com/logicorg/ivfs/finance/common/types/conversiontypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestRetrieveConversionRateHistory() {
	day := func(year int, month time.Month, d int) time.Time {
		return time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
	}
	var id int64
	rate := func(at time.Time) *model.Conversion {
		id++
		return &model.Conversion{
			ID: id, FromCurrencyID: 1, ToCurrencyID: 2, Rate: decimal.NewFromInt(id), ObservedAt: at,
		}
	}
	// daily is a rate at noon on each of n days ending on last, newest first,
	// as the activity answers.
	daily := func(last time.Time, n int) []*model.Conversion {
		rates := make([]*model.Conversion, n)
		for i := range rates {
			rates[i] = rate(last.AddDate(0, 0, -i).Add(12 * time.Hour))
		}
		return rates
	}
	reversed := func(rates ...*model.Conversion) []*model.Conversion {
		out := make([]*model.Conversion, 0, len(rates))
		for i := len(rates) - 1; i >= 0; i-- {
			out = append(out, rates[i])
		}
		return out
	}

	week := ct.RetrieveRateHistoryParams{FromCurrency: 1, ToCurrency: 2, From: day(2026, 10, 5), To: day(2026, 10, 11)}
	sunday, friday := rate(day(2026, 10, 11).Add(16*time.Hour)), rate(day(2026, 10, 9).Add(16*time.Hour))
	fridayMorning, saturdayBefore := rate(day(2026, 10, 9).Add(9*time.Hour)), rate(day(2026, 10, 3).Add(16*time.Hour))
	older := rate(day(2026, 10, 1).Add(16 * time.Hour))

	// A history longer than a page, whose first page cuts 5 July short: its
	// later rate is on the page, its earlier one is not asked for again.
	long := ct.RetrieveRateHistoryParams{FromCurrency: 1, ToCurrency: 2, From: day(2026, 1, 1), To: day(2026, 10, 11)}
	firstPage := append(daily(day(2026, 10, 11), ratePageSize-2),
		rate(day(2026, 7, 5).Add(18*time.Hour)), rate(day(2026, 7, 5).Add(9*time.Hour)))
	july4, newYearsEve := rate(day(2026, 7, 4).Add(12*time.Hour)), rate(day(2025, 12, 31).Add(12*time.Hour))
	cutShort := firstPage[:ratePageSize-1]

	type page struct {
		date  time.Time
		rates []*model.Conversion
		err   error
	}
	notFound := errors.ErrNotFound{ResourceType: "Conversion"}

	tests := map[string]struct {
		params  ct.RetrieveRateHistoryParams
		pages   []page
		want    []*model.Conversion
		wantErr string
	}{
		"each day's latest rate, with the one carried in": {
			params: week,
			pages:  []page{{date: day(2026, 10, 11), rates: []*model.Conversion{sunday, friday, fridayMorning, saturdayBefore, older}}},
			want:   []*model.Conversion{saturdayBefore, friday, sunday},
		},
		"nothing before the range": {
			params: week,
			pages:  []page{{date: day(2026, 10, 11), rates: []*model.Conversion{sunday, friday}}},
			want:   []*model.Conversion{friday, sunday},
		},
		"no rates at all": {
			params: week,
			pages:  []page{{date: day(2026, 10, 11), err: notFound}},
			want:   []*model.Conversion{},
		},
		"pages backwards from the day before a page's last one": {
			params: long,
			pages: []page{
				{date: day(2026, 10, 11), rates: firstPage},
				{date: day(2026, 7, 4), rates: []*model.Conversion{july4, newYearsEve}},
			},
			want: append([]*model.Conversion{newYearsEve, july4}, reversed(cutShort...)...),
		},
		"a page that ends the rates exactly": {
			params: long,
			pages: []page{
				{date: day(2026, 10, 11), rates: firstPage},
				{date: day(2026, 7, 4), err: notFound},
			},
			want: reversed(cutShort...),
		},
		"bounds in other zones, by their own dates": {
			params: ct.RetrieveRateHistoryParams{
				FromCurrency: 1, ToCurrency: 2,
				From: time.Date(2026, 10, 5, 1, 0, 0, 0, time.FixedZone("CEST", 2*60*60)),
				To:   time.Date(2026, 10, 11, 23, 0, 0, 0, time.FixedZone("EDT", -4*60*60)),
			},
			pages: []page{{date: day(2026, 10, 11), rates: []*model.Conversion{sunday, saturdayBefore, older}}},
			want:  []*model.Conversion{saturdayBefore, sunday},
		},
		"a page that fails": {
			params: long,
			pages: []page{
				{date: day(2026, 10, 11), rates: firstPage},
				{date: day(2026, 7, 4), err: errors.ErrMissingRequiredParameter{}},
			},
			wantErr: "retrieve the rates on or before 2026-07-04",
		},
		"a pair of the same currency, without calling the activity": {
			params:  ct.RetrieveRateHistoryParams{FromCurrency: 1, ToCurrency: 1, From: day(2026, 10, 5), To: day(2026, 10, 11)},
			wantErr: "the rate from currency 1 to itself is 1; ask for two different currencies",
		},
		"no currencies, without calling the activity": {
			params:  ct.RetrieveRateHistoryParams{From: day(2026, 10, 5), To: day(2026, 10, 11)},
			wantErr: "missing required parameters: FromCurrency, ToCurrency",
		},
		"a range that ends before it starts, without calling the activity": {
			params:  ct.RetrieveRateHistoryParams{FromCurrency: 1, ToCurrency: 2, From: day(2026, 10, 11), To: day(2026, 10, 5)},
			wantErr: "the history ends on 2026-10-05, before it starts on 2026-10-11",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			var ca *conversion.Conversion
			for _, p := range tc.pages {
				var response *ct.RetrieveResponse
				if p.err == nil {
					response = &ct.RetrieveResponse{Conversions: p.rates}
				}
				env.OnActivity(ca.RetrieveMostRecentCurrencyConversionRatesAtDate, mock.Anything, ct.RetrieveRequest{
					FromCurrency: 1, ToCurrency: 2, Date: p.date,
				}, int64(ratePageSize)).Return(response, p.err).Once()
			}

			env.ExecuteWorkflow(RetrieveConversionRateHistory, tc.params)

			env.AssertExpectations(s.T())
			err := env.GetWorkflowError()
			if tc.wantErr != "" {
				require.ErrorContains(s.T(), err, tc.wantErr)
				return
			}
			require.NoError(s.T(), err)
			var result *ct.RetrieveRateHistoryResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), &ct.RetrieveRateHistoryResult{Conversions: tc.want}, result)
		})
	}
}
