package institution

import (
	"time"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_Insert() {
	tests := map[string]struct {
		params it.InsertInstitutionParams
		// stored is the name expected in the database: trimmed, in the casing
		// it was given.
		stored string
		expErr bool
	}{
		"nominal": {
			params: it.InsertInstitutionParams{Name: "Revolut", Kind: "bank"},
			stored: "Revolut",
		},
		"a name with surrounding space": {
			params: it.InsertInstitutionParams{Name: "  Interactive Brokers  ", Kind: "broker"},
			stored: "Interactive Brokers",
		},
		"missing name": {
			params: it.InsertInstitutionParams{Kind: "bank"},
			expErr: true,
		},
		"missing kind": {
			params: it.InsertInstitutionParams{Name: "Revolut"},
			expErr: true,
		},
		"blank name": {
			params: it.InsertInstitutionParams{Name: "   ", Kind: "bank"},
			expErr: true,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.institution.Insert, tc.params)

			if tc.expErr {
				require.Error(s.T(), err)
				return
			}

			require.NoError(s.T(), err)

			var result it.InsertInstitutionResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.NotZero(s.T(), result.ID)

			var stored model.Institution
			require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", result.ID).Scan(s.T().Context()))
			require.Equal(s.T(), tc.stored, stored.Name)
			require.Equal(s.T(), tc.params.Kind, stored.Kind)

			temporaltesting.Delete(s.T(), s.DB, (*model.Institution)(nil), result.ID)
		})
	}
}

func (s *UnitTestSuite) Test_Insert_RetrieveOrInsertIdempotency() {
	params := it.InsertInstitutionParams{Name: "Revolut", Kind: "bank"}

	firstResult, err := s.Env.ExecuteActivity(s.institution.Insert, params)
	require.NoError(s.T(), err)
	var first it.InsertInstitutionResult
	require.NoError(s.T(), firstResult.Get(&first))

	secondResult, err := s.Env.ExecuteActivity(s.institution.Insert, params)
	require.NoError(s.T(), err)
	var second it.InsertInstitutionResult
	require.NoError(s.T(), secondResult.Get(&second))

	require.Equal(s.T(), first.ID, second.ID)

	var count int64
	count, err = s.DB.NewSelect().Model((*model.Institution)(nil)).Where("name = ?", params.Name).Count(s.T().Context())
	require.NoError(s.T(), err)
	require.EqualValues(s.T(), 1, count)

	temporaltesting.Delete(s.T(), s.DB, (*model.Institution)(nil), first.ID)
}

func (s *UnitTestSuite) Test_Insert_RetrievesExistingWithoutOverwriting() {
	openedAt := time.Now().UTC().Truncate(time.Second)
	existing := &model.Institution{Name: "Revolut", Kind: "bank", OpenedAt: &openedAt}
	temporaltesting.Insert(s.T(), s.DB, existing)

	// A retried insert with different (e.g. missing) OpenedAt must not
	// overwrite the existing row's data.
	activityResult, err := s.Env.ExecuteActivity(s.institution.Insert, it.InsertInstitutionParams{Name: "Revolut", Kind: "bank"})
	require.NoError(s.T(), err)
	var result it.InsertInstitutionResult
	require.NoError(s.T(), activityResult.Get(&result))
	require.Equal(s.T(), existing.ID, result.ID)

	var stored model.Institution
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", existing.ID).Scan(s.T().Context()))
	require.NotNil(s.T(), stored.OpenedAt)
	require.Equal(s.T(), openedAt, stored.OpenedAt.UTC())

	temporaltesting.Delete(s.T(), s.DB, (*model.Institution)(nil), existing.ID)
}

// Test_Insert_RetrievesExistingByName checks that an institution is retrieved
// by its name compared case-insensitively and trimmed (ADR 0005), and keeps
// the casing it was created with.
func (s *UnitTestSuite) Test_Insert_RetrievesExistingByName() {
	tests := map[string]string{
		"the same name":     "Revolut",
		"another casing":    "REVOLUT",
		"surrounding space": "  revolut  ",
	}

	for name, requested := range tests {
		s.Run(name, func() {
			existing := &model.Institution{Name: "Revolut", Kind: "bank"}
			temporaltesting.Insert(s.T(), s.DB, existing)
			s.T().Cleanup(func() { temporaltesting.Delete(s.T(), s.DB, (*model.Institution)(nil), existing.ID) })

			activityResult, err := s.Env.ExecuteActivity(s.institution.Insert, it.InsertInstitutionParams{Name: requested, Kind: "bank"})

			require.NoError(s.T(), err)
			var result it.InsertInstitutionResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.Equal(s.T(), existing.ID, result.ID)
			require.Equal(s.T(), 1, s.Count[model.Institution]())
			var stored model.Institution
			temporaltesting.Retrieve(s.T(), s.DB, &stored, existing.ID)
			require.Equal(s.T(), "Revolut", stored.Name)
		})
	}
}

type insertOutcome struct {
	result *it.InsertInstitutionResult
	err    error
}

// Test_Insert_RetryWaitsForConcurrentWriter checks that an insert racing
// another one for the same name, in another casing, returns the institution
// the other one wrote instead of creating a second: the lookup cannot see the
// uncommitted row, so it is the unique index on the name that stops the second
// insert until the first commits.
func (s *UnitTestSuite) Test_Insert_RetryWaitsForConcurrentWriter() {
	writer, err := s.DB.BeginTx(s.T().Context(), nil)
	require.NoError(s.T(), err)
	s.T().Cleanup(func() { _ = writer.Rollback() })
	first := &model.Institution{Name: "Revolut", Kind: "bank"}
	_, err = writer.NewInsert().Model(first).Returning("id").Exec(s.T().Context())
	require.NoError(s.T(), err)

	done := make(chan insertOutcome, 1)
	go func() {
		result, err := s.institution.Insert(s.T().Context(), it.InsertInstitutionParams{Name: "revolut", Kind: "bank"})
		done <- insertOutcome{result, err}
	}()
	select {
	case <-done:
		s.T().Fatal("the insert finished without waiting for the transaction writing the same name")
	case <-time.After(500 * time.Millisecond):
	}

	require.NoError(s.T(), writer.Commit())
	outcome := <-done
	require.NoError(s.T(), outcome.err)
	require.Equal(s.T(), first.ID, outcome.result.ID)
	require.Equal(s.T(), 1, s.Count[model.Institution]())
}
