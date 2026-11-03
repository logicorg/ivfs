package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_RenameCounterparty() {
	counterparty := &model.Counterparty{Name: "Tesco"}
	temporaltesting.Insert(s.T(), s.DB, counterparty)

	tests := map[string]struct {
		params  lt.RenameCounterpartyInput
		expErr  bool
		expName string
	}{
		"rename": {
			params:  lt.RenameCounterpartyInput{CounterpartyID: counterparty.ID, Name: "Tesco Ltd"},
			expName: "Tesco Ltd",
		},
		"blank name": {
			params: lt.RenameCounterpartyInput{CounterpartyID: counterparty.ID, Name: "   "},
			expErr: true,
		},
		"invalid id": {
			params: lt.RenameCounterpartyInput{CounterpartyID: 0, Name: "x"},
			expErr: true,
		},
		"not found": {
			params: lt.RenameCounterpartyInput{CounterpartyID: 999999, Name: "x"},
			expErr: true,
		},
	}

	for name, tc := range tests {
		temporaltesting.InsertOrRetrieve(s.T(), s.DB, counterparty, "id")

		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.RenameCounterparty, tc.params)
			if tc.expErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)

			var stored model.Counterparty
			require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", counterparty.ID).Scan(s.T().Context()))
			require.Equal(s.T(), tc.expName, stored.Name)
		})

		temporaltesting.Delete(s.T(), s.DB, (*model.Counterparty)(nil), counterparty.ID)
	}
}

func (s *UnitTestSuite) Test_RenameCounterparty_NotFound() {
	_, err := s.Env.ExecuteActivity(s.aa.RenameCounterparty, lt.RenameCounterpartyInput{
		CounterpartyID: 999999,
		Name:           "x",
	})
	require.Error(s.T(), err)
}

func (s *UnitTestSuite) Test_RenameCounterparty_Idempotency() {
	counterparty := &model.Counterparty{Name: "Tesco"}
	temporaltesting.Insert(s.T(), s.DB, counterparty)

	params := lt.RenameCounterpartyInput{CounterpartyID: counterparty.ID, Name: "Tesco Ltd"}

	_, err := s.Env.ExecuteActivity(s.aa.RenameCounterparty, params)
	require.NoError(s.T(), err)
	_, err = s.Env.ExecuteActivity(s.aa.RenameCounterparty, params)
	require.NoError(s.T(), err)

	var stored model.Counterparty
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", counterparty.ID).Scan(s.T().Context()))
	require.Equal(s.T(), "Tesco Ltd", stored.Name)
}

// Test_RenameCounterparty_Taken checks that a counterparty cannot take the
// name of another, compared case-insensitively (ADR 0005), and is left as it
// was, while renaming it to its own name in another casing is allowed.
func (s *UnitTestSuite) Test_RenameCounterparty_Taken() {
	tests := map[string]struct {
		// target picks the counterparty to rename, of Tesco and Waitrose.
		target  func(tesco, waitrose *model.Counterparty) *model.Counterparty
		newName string
		errStr  string
		stored  string
	}{
		"the name of another counterparty": {
			target:  func(_, waitrose *model.Counterparty) *model.Counterparty { return waitrose },
			newName: " tesco ",
			errStr:  "resource Counterparty conflict",
			stored:  "Waitrose",
		},
		"its own name in another casing": {
			target:  func(tesco, _ *model.Counterparty) *model.Counterparty { return tesco },
			newName: "TESCO",
			stored:  "TESCO",
		},
	}

	for name, tc := range tests {
		tesco := &model.Counterparty{Name: "Tesco"}
		temporaltesting.Insert(s.T(), s.DB, tesco)
		waitrose := &model.Counterparty{Name: "Waitrose"}
		temporaltesting.Insert(s.T(), s.DB, waitrose)
		target := tc.target(tesco, waitrose)

		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.RenameCounterparty, lt.RenameCounterpartyInput{CounterpartyID: target.ID, Name: tc.newName})

			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
			} else {
				require.NoError(s.T(), err)
			}
			var stored model.Counterparty
			temporaltesting.Retrieve(s.T(), s.DB, &stored, target.ID)
			require.Equal(s.T(), tc.stored, stored.Name)
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Counterparty)(nil), tesco.ID)
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Counterparty)(nil), waitrose.ID)
	}
}
