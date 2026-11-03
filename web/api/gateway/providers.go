package gateway

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/logicorg/ivfs/pkg/assetvalue"
)

// RangeInput is the dates an endpoint over the asset value providers asks
// them for (ADR 0014).
type RangeInput struct {
	From     string `query:"from" format:"date" required:"true" doc:"the first date, such as 2026-10-01"`
	To       string `query:"to" format:"date" required:"true" doc:"the last date, included; from itself for a single day"`
	Interval string `query:"interval" enum:"day,week,month" default:"day" doc:"one value a day, a week, dated its Sunday, or a month, dated its last day; the last value is dated to"`
}

// Range returns in as the providers take it, or a problem when it is not a
// range.
func (in RangeInput) Range() (assetvalue.Range, error) {
	from, err := time.Parse(time.DateOnly, in.From)
	if err != nil {
		return assetvalue.Range{}, huma.Error422UnprocessableEntity("from is not a date, such as 2026-10-01")
	}
	to, err := time.Parse(time.DateOnly, in.To)
	if err != nil {
		return assetvalue.Range{}, huma.Error422UnprocessableEntity("to is not a date, such as 2026-10-10")
	}
	r := assetvalue.Range{From: from, To: to, Interval: assetvalue.Interval(in.Interval)}
	// Checked before any provider is asked, so a bad range is one problem
	// rather than a failure of every provider.
	if _, err := r.Dates(); err != nil {
		return assetvalue.Range{}, huma.Error422UnprocessableEntity(err.Error())
	}
	return r, nil
}

// ProviderError is a provider that did not answer, so the others still can.
type ProviderError struct {
	Provider string `json:"provider"`
	Detail   string `json:"detail" doc:"why, such as no answer within 5s"`
}
