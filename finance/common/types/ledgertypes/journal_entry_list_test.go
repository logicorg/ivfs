package ledgertypes

import (
	"testing"
	"time"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestListJournalEntriesParams_Validate(t *testing.T) {
	day := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	const (
		accepted   = ""
		constraint = "cue"
		rule       = "rule"
	)

	tests := map[string]struct {
		params ListJournalEntriesParams
		// failure is how the parameters are rejected: by a CUE constraint, which is
		// a ValidationError, or by a rule CUE cannot express, which is an
		// ErrInvalidParameter.
		failure string
	}{
		"no filter":                          {params: ListJournalEntriesParams{}},
		"every filter":                       {params: ListJournalEntriesParams{LedgerAccountID: 1, CounterpartyID: 2, ExternalID: "bank-42", Statuses: []model.JournalEntryStatus{model.JournalEntryPosted}, From: day, To: day.Add(time.Hour)}},
		"several statuses":                   {params: ListJournalEntriesParams{Statuses: []model.JournalEntryStatus{model.JournalEntryDraft, model.JournalEntryPosted}}},
		"newest first":                       {params: ListJournalEntriesParams{Descending: true}},
		"a range with no start":              {params: ListJournalEntriesParams{To: day}},
		"a range with no end":                {params: ListJournalEntriesParams{From: day}},
		"the largest page":                   {params: ListJournalEntriesParams{Limit: MaxJournalEntriesLimit}},
		"a cursor":                           {params: ListJournalEntriesParams{After: &JournalEntryCursor{OccurredAt: day, ID: 3}}},
		"a negative account":                 {failure: constraint, params: ListJournalEntriesParams{LedgerAccountID: -1}},
		"a negative counterparty":            {failure: constraint, params: ListJournalEntriesParams{CounterpartyID: -1}},
		"a negative limit":                   {failure: constraint, params: ListJournalEntriesParams{Limit: -1}},
		"a page that is too large":           {failure: rule, params: ListJournalEntriesParams{Limit: MaxJournalEntriesLimit + 1}},
		"a range that ends where it starts":  {failure: rule, params: ListJournalEntriesParams{From: day, To: day}},
		"a range that ends before it starts": {failure: rule, params: ListJournalEntriesParams{From: day, To: day.Add(-time.Hour)}},
		"an unknown status":                  {failure: rule, params: ListJournalEntriesParams{Statuses: []model.JournalEntryStatus{model.JournalEntryPosted, "archived"}}},
		"a cursor without an id":             {failure: rule, params: ListJournalEntriesParams{After: &JournalEntryCursor{OccurredAt: day}}},
		"a cursor without a date":            {failure: rule, params: ListJournalEntriesParams{After: &JournalEntryCursor{ID: 3}}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.params.Validate()

			switch tc.failure {
			case accepted:
				require.NoError(t, err)
			case constraint:
				require.ErrorAs(t, err, &internalerrors.ValidationError{})
			case rule:
				require.ErrorAs(t, err, &internalerrors.ErrInvalidParameter{})
			}
		})
	}
}

func TestListJournalEntriesParams_PageSize(t *testing.T) {
	require.Equal(t, DefaultJournalEntriesLimit, (&ListJournalEntriesParams{}).PageSize())
	require.Equal(t, 7, (&ListJournalEntriesParams{Limit: 7}).PageSize())
}
