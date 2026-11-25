package accounts

import (
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
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
		params  act.RetrieveFinancialAccountParams
		expErr  bool
		expName string
	}{
		"found": {
			params:  act.RetrieveFinancialAccountParams{ID: account.ID},
			expName: "Revolut EUR",
		},
		"invalid id": {
			params: act.RetrieveFinancialAccountParams{ID: 0},
			expErr: true,
		},
		"not found": {
			params: act.RetrieveFinancialAccountParams{ID: 999999},
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

			var result act.RetrieveFinancialAccountResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.NotNil(s.T(), result.FinancialAccount)
			require.Equal(s.T(), tc.expName, result.FinancialAccount.Name)
		})
	}
}
