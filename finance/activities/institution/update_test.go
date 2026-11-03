package institution

import (
	"testing"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_Update() {
	tcs := map[string]struct {
		// findBy selects the institution to update, given the fixture.
		findBy  func(institution *model.Institution) it.RetrieveInstitutionParams
		name    string
		expErr  bool
		expName string
	}{
		"by id": {
			findBy: func(institution *model.Institution) it.RetrieveInstitutionParams {
				return it.RetrieveInstitutionParams{ID: institution.ID}
			},
			name:    "investments",
			expName: "investments",
		},
		"by name": {
			findBy: func(institution *model.Institution) it.RetrieveInstitutionParams {
				return it.RetrieveInstitutionParams{Name: institution.Name}
			},
			name:    "savings",
			expName: "savings",
		},
		"missing find by": {
			findBy: func(*model.Institution) it.RetrieveInstitutionParams { return it.RetrieveInstitutionParams{} },
			name:   "invalid",
			expErr: true,
		},
	}

	for name, tc := range tcs {
		institution := &model.Institution{Name: "cash"}
		temporaltesting.Insert(s.T(), s.DB, institution)

		s.T().Run(name, func(t *testing.T) {
			res, err := s.Env.ExecuteActivity(s.institution.Update, it.UpdateInstitutionParams{FindBy: tc.findBy(institution), Name: tc.name})
			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			var activityResult it.UpdateInstitutionResult
			err = res.Get(&activityResult)
			require.NoError(t, err)
			require.Equal(t, institution.ID, activityResult.ID)

			updated := &model.Institution{}
			temporaltesting.Retrieve(t, s.DB, updated, institution.ID)
			require.Equal(t, tc.expName, updated.Name)
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Institution)(nil), institution.ID)
	}
}

// Test_Update_Taken checks that an institution cannot take the name of
// another, compared case-insensitively (ADR 0005), and is left as it was,
// while renaming it to its own name in another casing is allowed, and a new
// name is stored trimmed.
func (s *UnitTestSuite) Test_Update_Taken() {
	tests := map[string]struct {
		// target picks the institution to rename, of Revolut and N26.
		target  func(revolut, n26 *model.Institution) *model.Institution
		newName string
		errStr  string
		stored  string
	}{
		"the name of another institution": {
			target:  func(_, n26 *model.Institution) *model.Institution { return n26 },
			newName: "revolut",
			errStr:  "resource Institution conflict",
			stored:  "N26",
		},
		"its own name in another casing": {
			target:  func(revolut, _ *model.Institution) *model.Institution { return revolut },
			newName: "REVOLUT",
			stored:  "REVOLUT",
		},
		"a name with surrounding space": {
			target:  func(_, n26 *model.Institution) *model.Institution { return n26 },
			newName: "  N26 Bank  ",
			stored:  "N26 Bank",
		},
	}

	for name, tc := range tests {
		revolut := &model.Institution{Name: "Revolut", Kind: "bank"}
		temporaltesting.Insert(s.T(), s.DB, revolut)
		n26 := &model.Institution{Name: "N26", Kind: "bank"}
		temporaltesting.Insert(s.T(), s.DB, n26)
		target := tc.target(revolut, n26)

		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.institution.Update, it.UpdateInstitutionParams{
				FindBy: it.RetrieveInstitutionParams{ID: target.ID}, Name: tc.newName,
			})

			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
			} else {
				require.NoError(s.T(), err)
			}
			var stored model.Institution
			temporaltesting.Retrieve(s.T(), s.DB, &stored, target.ID)
			require.Equal(s.T(), tc.stored, stored.Name)
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Institution)(nil), revolut.ID)
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Institution)(nil), n26.ID)
	}
}
