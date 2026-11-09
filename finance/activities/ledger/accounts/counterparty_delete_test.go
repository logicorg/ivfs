package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_DeleteCounterparty() {
	fixture := newFixture(s)
	const stillReferenced = "resource Counterparty conflict: still referenced by ledger.journal_entry.counterparty_id"

	tests := map[string]struct {
		// posted and drafts are the entries that still name the counterparty.
		posted, drafts int
		errStr         string
	}{
		"unreferenced":                 {},
		"referenced by a posted entry": {posted: 1, errStr: stillReferenced},
		"referenced by a draft":        {drafts: 1, errStr: stillReferenced},
	}

	for name, tc := range tests {
		payee := &model.Counterparty{Name: "Tesco"}
		temporaltesting.Insert(s.T(), s.DB, payee)
		var entries []*model.JournalEntry
		for range tc.posted {
			entries = append(entries, fixture.post(payee.ID))
		}
		for range tc.drafts {
			entries = append(entries, fixture.draft(payee.ID))
		}

		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.DeleteCounterparty, lt.DeleteCounterpartyInput{CounterpartyID: payee.ID})

			exists, existsErr := s.DB.NewSelect().Model((*model.Counterparty)(nil)).Where("id = ?", payee.ID).Exists(s.T().Context())
			require.NoError(s.T(), existsErr)
			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
				require.True(s.T(), exists, "a refused delete must leave the counterparty in place")
				return
			}
			require.NoError(s.T(), err)
			require.False(s.T(), exists)
		})

		fixture.deleteAll(entries)
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Counterparty)(nil), payee.ID)
	}
}

func (s *UnitTestSuite) Test_DeleteCounterparty_Input() {
	tests := map[string]struct {
		id     int64
		errStr string
	}{
		"missing counterparty": {id: 999999},
		"invalid id":           {id: 0, errStr: "validation error"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.DeleteCounterparty, lt.DeleteCounterpartyInput{CounterpartyID: tc.id})
			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
				return
			}
			require.NoError(s.T(), err)
		})
	}
}

// Test_DeleteCounterparty_FreesName checks that the name of a deleted
// counterparty, compared case-insensitively, can be taken again (ADR 0016).
func (s *UnitTestSuite) Test_DeleteCounterparty_FreesName() {
	payee := &model.Counterparty{Name: "TESCO"}
	temporaltesting.Insert(s.T(), s.DB, payee)

	_, err := s.Env.ExecuteActivity(s.aa.DeleteCounterparty, lt.DeleteCounterpartyInput{CounterpartyID: payee.ID})
	require.NoError(s.T(), err)

	again := &model.Counterparty{Name: "Tesco"}
	temporaltesting.Insert(s.T(), s.DB, again)
	temporaltesting.Delete(s.T(), s.DB, (*model.Counterparty)(nil), again.ID)
}

func (s *UnitTestSuite) Test_DeleteCounterparty_Idempotency() {
	payee := &model.Counterparty{Name: "Tesco"}
	temporaltesting.Insert(s.T(), s.DB, payee)
	input := lt.DeleteCounterpartyInput{CounterpartyID: payee.ID}

	for range 2 {
		_, err := s.Env.ExecuteActivity(s.aa.DeleteCounterparty, input)
		require.NoError(s.T(), err)
	}

	exists, err := s.DB.NewSelect().Model((*model.Counterparty)(nil)).Where("id = ?", payee.ID).Exists(s.T().Context())
	require.NoError(s.T(), err)
	require.False(s.T(), exists)
}
