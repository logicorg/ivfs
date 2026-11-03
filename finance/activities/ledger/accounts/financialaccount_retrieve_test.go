package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_RetrieveFinancialAccount() {
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut EUR"}
	temporaltesting.Insert(s.T(), s.DB, account)

	tests := map[string]struct {
		params  lt.RetrieveFinancialAccountParams
		expErr  bool
		expName string
	}{
		"found": {
			params:  lt.RetrieveFinancialAccountParams{ID: account.ID},
			expName: "Revolut EUR",
		},
		"invalid id": {
			params: lt.RetrieveFinancialAccountParams{ID: 0},
			expErr: true,
		},
		"not found": {
			params: lt.RetrieveFinancialAccountParams{ID: 999999},
			expErr: true,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.RetrieveFinancialAccount, tc.params)
			if tc.expErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)

			var result lt.RetrieveFinancialAccountResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.NotNil(s.T(), result.FinancialAccount)
			require.Equal(s.T(), tc.expName, result.FinancialAccount.Name)
		})
	}
}
