package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) TestDeleteLedgerAccount() {
	f := insertBalanceFixture(s.T(), s)

	tests := map[string]struct {
		input  lt.DeleteLedgerAccountInput
		errStr string
	}{
		"an asset with a balance": {
			input:  lt.DeleteLedgerAccountInput{LedgerAccountID: f.cash.ID},
			errStr: "resource LedgerAccount conflict: ledger account has postings; close it instead",
		},
		"an asset with a draft entry": {
			input:  lt.DeleteLedgerAccountInput{LedgerAccountID: f.drafted.ID},
			errStr: "resource LedgerAccount conflict: ledger account has postings; close it instead",
		},
		"an account with no postings": {
			input: lt.DeleteLedgerAccountInput{LedgerAccountID: f.unused.ID},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			t := s.T()

			_, err := s.Env.ExecuteActivity(s.aa.DeleteLedgerAccount, tc.input)

			exists, errExists := s.DB.NewSelect().Model((*model.LedgerAccount)(nil)).Where("id = ?", tc.input.LedgerAccountID).Exists(t.Context())
			require.NoError(t, errExists, "querying for existence of ledger account")

			if tc.errStr != "" {
				require.ErrorContains(t, err, tc.errStr)
				require.True(t, exists, "ledger account should still exist")
				return
			}

			require.NoError(t, err)
			require.False(t, exists, "ledger account should have been deleted")
		})
	}
}
