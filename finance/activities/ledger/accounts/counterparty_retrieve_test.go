package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_RetrieveCounterparty() {
	counterparty := &model.Counterparty{Name: "Tesco"}
	temporaltesting.Insert(s.T(), s.DB, counterparty)

	tests := map[string]struct {
		params  lt.RetrieveCounterpartyParams
		expErr  bool
		expName string
	}{
		"found": {
			params:  lt.RetrieveCounterpartyParams{ID: counterparty.ID},
			expName: "Tesco",
		},
		"found by name": {
			params:  lt.RetrieveCounterpartyParams{Name: "Tesco"},
			expName: "Tesco",
		},
		"found by name in another casing": {
			params:  lt.RetrieveCounterpartyParams{Name: "TESCO"},
			expName: "Tesco",
		},
		"invalid id": {
			params: lt.RetrieveCounterpartyParams{ID: 0},
			expErr: true,
		},
		"not found": {
			params: lt.RetrieveCounterpartyParams{ID: 999999},
			expErr: true,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.RetrieveCounterparty, tc.params)
			if tc.expErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)

			var result lt.RetrieveCounterpartyResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.NotNil(s.T(), result.Counterparty)
			require.Equal(s.T(), tc.expName, result.Counterparty.Name)
		})
	}
}

func (s *UnitTestSuite) Test_ListCounterparties() {
	tesco := &model.Counterparty{Name: "Tesco"}
	temporaltesting.Insert(s.T(), s.DB, tesco)
	waitrose := &model.Counterparty{Name: "Waitrose"}
	temporaltesting.Insert(s.T(), s.DB, waitrose)
	acme := &model.Counterparty{Name: "Acme Employer"}
	temporaltesting.Insert(s.T(), s.DB, acme)

	tests := map[string]struct {
		params lt.ListCounterpartiesParams
		expIDs []int64
	}{
		"no filter lists all": {
			params: lt.ListCounterpartiesParams{},
			expIDs: []int64{tesco.ID, waitrose.ID, acme.ID},
		},
		"search matches substring case-insensitively": {
			params: lt.ListCounterpartiesParams{NameContains: "tesc"},
			expIDs: []int64{tesco.ID},
		},
		"search with no match returns empty": {
			params: lt.ListCounterpartiesParams{NameContains: "nonexistent"},
			expIDs: []int64{},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.ListCounterparties, tc.params)
			require.NoError(s.T(), err)

			var result lt.ListCounterpartiesResult
			require.NoError(s.T(), activityResult.Get(&result))
			gotIDs := make([]int64, 0, len(result.Counterparties))
			for _, cp := range result.Counterparties {
				gotIDs = append(gotIDs, cp.ID)
			}
			require.ElementsMatch(s.T(), tc.expIDs, gotIDs)
		})
	}
}
