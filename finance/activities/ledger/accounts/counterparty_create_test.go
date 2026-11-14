package accounts

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) createCounterparty(input lt.CreateCounterpartyInput) lt.CreateCounterpartyResult {
	s.T().Helper()

	activityResult, err := s.Env.ExecuteActivity(s.aa.CreateCounterparty, input)
	require.NoError(s.T(), err)
	var result lt.CreateCounterpartyResult
	require.NoError(s.T(), activityResult.Get(&result))
	return result
}

// Test_CreateCounterparty checks that a counterparty is stored with its name
// trimmed but in the casing it was given, and that a blank name is rejected.
func (s *UnitTestSuite) Test_CreateCounterparty() {
	tests := map[string]struct {
		name   string
		stored string
		errStr string
	}{
		"a name":                        {name: "Tesco", stored: "Tesco"},
		"a name with surrounding space": {name: "  Waitrose  ", stored: "Waitrose"},
		"a blank name":                  {name: "   ", errStr: "missing required parameters: Name"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.CreateCounterparty, lt.CreateCounterpartyInput{Name: tc.name})

			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
				require.Zero(s.T(), s.Count[model.Counterparty]())
				return
			}
			require.NoError(s.T(), err)
			var result lt.CreateCounterpartyResult
			require.NoError(s.T(), activityResult.Get(&result))
			s.T().Cleanup(func() { temporaltesting.Delete(s.T(), s.DB, (*model.Counterparty)(nil), result.CounterpartyID) })

			var stored model.Counterparty
			temporaltesting.Retrieve(s.T(), s.DB, &stored, result.CounterpartyID)
			require.Equal(s.T(), tc.stored, stored.Name)
		})
	}
}

// Test_CreateCounterparty_Idempotent checks that a counterparty is retrieved
// by its name, compared case-insensitively: a retry, or a create of the same
// name in another casing, returns the first counterparty and changes nothing,
// whatever the rest of the request says.
func (s *UnitTestSuite) Test_CreateCounterparty_Idempotent() {
	created := s.createCounterparty(lt.CreateCounterpartyInput{Name: "Tesco", ExternalID: ptr.NewPtr("merchant-1")})
	retried := s.createCounterparty(lt.CreateCounterpartyInput{Name: "Tesco", ExternalID: ptr.NewPtr("merchant-1")})

	require.Equal(s.T(), created, retried)
	require.Equal(s.T(), 1, s.Count[model.Counterparty]())

	different := s.createCounterparty(lt.CreateCounterpartyInput{Name: " TESCO ", ExternalID: ptr.NewPtr("merchant-2")})

	require.Equal(s.T(), created, different)
	require.Equal(s.T(), 1, s.Count[model.Counterparty]())
	var stored model.Counterparty
	temporaltesting.Retrieve(s.T(), s.DB, &stored, created.CounterpartyID)
	require.Equal(s.T(), "Tesco", stored.Name)
	require.Equal(s.T(), "merchant-1", *stored.ExternalID)
}

type counterpartyOutcome struct {
	result *lt.CreateCounterpartyResult
	err    error
}

// Test_CreateCounterparty_RetryWaitsForConcurrentWriter checks that a create
// racing another one for the same name, in another casing, returns the
// counterparty the other one wrote instead of creating a second: the lookup
// cannot see the uncommitted row, so it is the unique index on the name that
// stops the second insert until the first commits.
func (s *UnitTestSuite) Test_CreateCounterparty_RetryWaitsForConcurrentWriter() {
	writer, err := s.DB.BeginTx(s.T().Context(), nil)
	require.NoError(s.T(), err)
	s.T().Cleanup(func() { _ = writer.Rollback() })
	first := &model.Counterparty{Name: "Tesco"}
	_, err = writer.NewInsert().Model(first).Returning("id").Exec(s.T().Context())
	require.NoError(s.T(), err)

	done := make(chan counterpartyOutcome, 1)
	go func() {
		result, err := s.aa.CreateCounterparty(s.T().Context(), lt.CreateCounterpartyInput{Name: "tesco"})
		done <- counterpartyOutcome{result, err}
	}()
	select {
	case <-done:
		s.T().Fatal("the create finished without waiting for the transaction writing the same name")
	case <-time.After(500 * time.Millisecond):
	}

	require.NoError(s.T(), writer.Commit())
	outcome := <-done
	require.NoError(s.T(), outcome.err)
	require.Equal(s.T(), first.ID, outcome.result.CounterpartyID)
	require.Equal(s.T(), 1, s.Count[model.Counterparty]())
}
