package settings

import (
	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_RetrieveUserSettings() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)

	usd := testhelpers.USDollar()
	temporaltesting.Insert(s.T(), s.DB, usd)

	tests := map[string]struct {
		// stored is the row in the database before the read, given the
		// suite's currencies, or nil for none.
		stored            *model.UserSettings
		wantCurrencyID    *int64
		wantGapPolicy     model.GapPolicy
		wantStoredRowRead bool
	}{
		"never changed": {
			stored:         nil,
			wantCurrencyID: nil,
			wantGapPolicy:  model.GapPolicyExclude,
		},
		"a default currency": {
			stored:            &model.UserSettings{ID: model.UserSettingsID, DefaultCurrencyID: &usd.ID, GapPolicy: model.GapPolicyExclude},
			wantCurrencyID:    &usd.ID,
			wantGapPolicy:     model.GapPolicyExclude,
			wantStoredRowRead: true,
		},
		"stored without a default currency": {
			stored:            &model.UserSettings{ID: model.UserSettingsID, GapPolicy: model.GapPolicyExclude},
			wantCurrencyID:    nil,
			wantGapPolicy:     model.GapPolicyExclude,
			wantStoredRowRead: true,
		},
	}

	for name, tc := range tests {
		if tc.stored != nil {
			temporaltesting.Insert(s.T(), s.DB, tc.stored)
		}

		s.Run(name, func() {
			res, err := s.Env.ExecuteActivity(s.activity.RetrieveUserSettings)
			require.NoError(s.T(), err)

			var result st.RetrieveUserSettingsResult
			require.NoError(s.T(), res.Get(&result))
			require.Equal(s.T(), model.UserSettingsID, result.Settings.ID)
			require.Equal(s.T(), tc.wantCurrencyID, result.Settings.DefaultCurrencyID)
			require.Equal(s.T(), tc.wantGapPolicy, result.Settings.GapPolicy)
			// The defaults are never written: only the stored row has timestamps.
			require.Equal(s.T(), tc.wantStoredRowRead, !result.Settings.CreatedAt.IsZero())
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.UserSettings)(nil), model.UserSettingsID)
	}
}
