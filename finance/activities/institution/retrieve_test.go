package institution

import (
	"testing"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_Retrieve() {
	wrapper := &model.Institution{Name: "cash"}
	temporaltesting.Insert(s.T(), s.DB, wrapper)

	tcs := map[string]struct {
		params  it.RetrieveInstitutionParams
		expErr  bool
		expName string
	}{
		"by id": {
			params:  it.RetrieveInstitutionParams{ID: wrapper.ID},
			expName: wrapper.Name,
		},
		"by name": {
			params:  it.RetrieveInstitutionParams{Name: wrapper.Name},
			expName: wrapper.Name,
		},
		"by name in another casing": {
			params:  it.RetrieveInstitutionParams{Name: " CASH "},
			expName: wrapper.Name,
		},
		"missing input": {
			params: it.RetrieveInstitutionParams{},
			expErr: true,
		},
		"not found": {
			params: it.RetrieveInstitutionParams{ID: 999},
			expErr: true,
		},
	}

	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			res, err := s.Env.ExecuteActivity(s.institution.Retrieve, tc.params)
			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			var activityResult it.RetrieveInstitutionResult
			err = res.Get(&activityResult)
			require.NoError(t, err)
			require.NotNil(t, activityResult.Institution)
			require.Equal(t, tc.expName, activityResult.Institution.Name)
		})
	}
}
