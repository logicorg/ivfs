package institution

import (
	"database/sql"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_Delete() {
	institution := &model.Institution{Name: "cash"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	tcs := map[string]struct {
		params it.DeleteInstitutionParams
		expErr bool
	}{
		"by id": {
			params: it.DeleteInstitutionParams{
				FindBy: it.RetrieveInstitutionParams{ID: institution.ID},
			},
		},
		"by name": {
			params: it.DeleteInstitutionParams{
				FindBy: it.RetrieveInstitutionParams{Name: institution.Name},
			},
		},
		"missing input": {
			params: it.DeleteInstitutionParams{},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		temporaltesting.InsertOrRetrieve(s.T(), s.DB, institution, "id")

		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.institution.Delete, tc.params)

			if tc.expErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)

				err = s.DB.NewSelect().Model((*model.Institution)(nil)).Where("id = ?", institution.ID).Scan(s.T().Context())
				require.ErrorIs(s.T(), err, sql.ErrNoRows)
			}

		})

		temporaltesting.Delete(s.T(), s.DB, (*model.Institution)(nil), institution.ID)
	}
}

func (s *UnitTestSuite) Test_Delete_NotFoundIsNotAnError() {
	_, err := s.Env.ExecuteActivity(s.institution.Delete, it.DeleteInstitutionParams{
		FindBy: it.RetrieveInstitutionParams{ID: 999999},
	})
	require.NoError(s.T(), err)
}

func (s *UnitTestSuite) Test_Delete_Idempotency() {
	institution := &model.Institution{Name: "cash"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	params := it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: institution.ID}}

	_, err := s.Env.ExecuteActivity(s.institution.Delete, params)
	require.NoError(s.T(), err)
	_, err = s.Env.ExecuteActivity(s.institution.Delete, params)
	require.NoError(s.T(), err)

	err = s.DB.NewSelect().Model((*model.Institution)(nil)).Where("id = ?", institution.ID).Scan(s.T().Context())
	require.ErrorIs(s.T(), err, sql.ErrNoRows)
}

// An institution that financial accounts refer to cannot be deleted. The
// caller has to get a conflict that Temporal does not retry, and the
// institution must survive.
func (s *UnitTestSuite) Test_Delete_InstitutionInUse() {
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	temporaltesting.Insert(s.T(), s.DB, &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut EUR"})

	_, err := s.Env.ExecuteActivity(s.institution.Delete, it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: institution.ID}})

	var appErr *temporal.ApplicationError
	require.ErrorAs(s.T(), err, &appErr)
	require.Equal(s.T(), "ErrResourceConflict", appErr.Type())
	require.ErrorContains(s.T(), err, "still referenced")

	exists, err := s.DB.NewSelect().Model((*model.Institution)(nil)).Where("id = ?", institution.ID).Exists(s.T().Context())
	require.NoError(s.T(), err)
	require.True(s.T(), exists)
}
