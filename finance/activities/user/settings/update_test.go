package settings

import (
	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_UpdateUserSettings() {
	missing := int64(999999)

	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)

	usd := testhelpers.USDollar()
	temporaltesting.Insert(s.T(), s.DB, usd)

	tests := map[string]struct {
		// stored is the row in the database before the update, given the
		// suite's currencies, or nil for none.
		stored *model.UserSettings
		params st.UpdateUserSettingsParams
		// want is the row expected afterwards, or nil for no row.
		want      *model.UserSettings
		wantError string
	}{
		"the first update creates the row": {
			stored: nil,
			params: st.UpdateUserSettingsParams{DefaultCurrencyID: &eur.ID, GapPolicy: model.GapPolicyExclude},
			want:   &model.UserSettings{DefaultCurrencyID: &eur.ID, GapPolicy: model.GapPolicyExclude},
		},
		"a change of default currency": {
			stored: &model.UserSettings{ID: model.UserSettingsID, DefaultCurrencyID: &eur.ID, GapPolicy: model.GapPolicyExclude},
			params: st.UpdateUserSettingsParams{DefaultCurrencyID: &usd.ID, GapPolicy: model.GapPolicyExclude},
			want:   &model.UserSettings{DefaultCurrencyID: &usd.ID, GapPolicy: model.GapPolicyExclude},
		},
		"no default currency clears it": {
			stored: &model.UserSettings{ID: model.UserSettingsID, DefaultCurrencyID: &eur.ID, GapPolicy: model.GapPolicyExclude},
			params: st.UpdateUserSettingsParams{GapPolicy: model.GapPolicyExclude},
			want:   &model.UserSettings{GapPolicy: model.GapPolicyExclude},
		},
		"a currency that does not exist": {
			stored:    nil,
			params:    st.UpdateUserSettingsParams{DefaultCurrencyID: &missing, GapPolicy: model.GapPolicyExclude},
			want:      nil,
			wantError: "resource Currency not found with ID: 999999",
		},
		"the reserved cost basis policy leaves the row as it was": {
			stored:    &model.UserSettings{ID: model.UserSettingsID, DefaultCurrencyID: &eur.ID, GapPolicy: model.GapPolicyExclude},
			params:    st.UpdateUserSettingsParams{DefaultCurrencyID: &usd.ID, GapPolicy: model.GapPolicyCostBasis},
			want:      &model.UserSettings{DefaultCurrencyID: &eur.ID, GapPolicy: model.GapPolicyExclude},
			wantError: "the gap policy cost_basis is not available yet; use exclude",
		},
		"no gap policy": {
			stored:    nil,
			params:    st.UpdateUserSettingsParams{DefaultCurrencyID: &eur.ID},
			want:      nil,
			wantError: "missing required parameters: GapPolicy",
		},
	}

	for name, tc := range tests {
		if tc.stored != nil {
			temporaltesting.Insert(s.T(), s.DB, tc.stored)
		}

		s.Run(name, func() {
			res, err := s.Env.ExecuteActivity(s.activity.UpdateUserSettings, tc.params)
			want := tc.want
			if tc.wantError != "" {
				require.ErrorContains(s.T(), err, tc.wantError)
			} else {
				require.NoError(s.T(), err)
				var result st.UpdateUserSettingsResult
				require.NoError(s.T(), res.Get(&result))
				require.Equal(s.T(), want.DefaultCurrencyID, result.Settings.DefaultCurrencyID)
				require.Equal(s.T(), want.GapPolicy, result.Settings.GapPolicy)
				require.False(s.T(), result.Settings.CreatedAt.IsZero())
			}

			var rows []model.UserSettings
			require.NoError(s.T(), s.DB.NewSelect().Model(&rows).Scan(s.T().Context()))
			if want == nil {
				require.Empty(s.T(), rows)
				return
			}
			require.Len(s.T(), rows, 1)
			require.Equal(s.T(), want.DefaultCurrencyID, rows[0].DefaultCurrencyID)
			require.Equal(s.T(), want.GapPolicy, rows[0].GapPolicy)
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.UserSettings)(nil), model.UserSettingsID)
	}
}

// Test_UpdateUserSettings_Idempotency runs the same update twice, as a retry
// would, and checks it leaves one row with the settings, created once.
func (s *UnitTestSuite) Test_UpdateUserSettings_Idempotency() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)

	usd := testhelpers.USDollar()
	temporaltesting.Insert(s.T(), s.DB, usd)

	params := st.UpdateUserSettingsParams{DefaultCurrencyID: &eur.ID, GapPolicy: model.GapPolicyExclude}

	res, err := s.Env.ExecuteActivity(s.activity.UpdateUserSettings, params)
	require.NoError(s.T(), err)
	var first st.UpdateUserSettingsResult
	require.NoError(s.T(), res.Get(&first))

	res, err = s.Env.ExecuteActivity(s.activity.UpdateUserSettings, params)
	require.NoError(s.T(), err)
	var second st.UpdateUserSettingsResult
	require.NoError(s.T(), res.Get(&second))

	require.Equal(s.T(), first.Settings.DefaultCurrencyID, second.Settings.DefaultCurrencyID)
	require.Equal(s.T(), first.Settings.GapPolicy, second.Settings.GapPolicy)
	require.True(s.T(), first.Settings.CreatedAt.Equal(second.Settings.CreatedAt))

	var rows []model.UserSettings
	require.NoError(s.T(), s.DB.NewSelect().Model(&rows).Scan(s.T().Context()))
	require.Len(s.T(), rows, 1)
	require.Equal(s.T(), &eur.ID, rows[0].DefaultCurrencyID)
	require.Equal(s.T(), model.GapPolicyExclude, rows[0].GapPolicy)
	require.True(s.T(), first.Settings.CreatedAt.Equal(rows[0].CreatedAt))
}
