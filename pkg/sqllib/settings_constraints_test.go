package sqllib_test

import (
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

// Test_UserSettingsConstraints checks the rules of the settings row that the
// database enforces, running the inserts directly, as anything other than the
// activities would.
func (s *UnitTestSuite) Test_UserSettingsConstraints() {
	tests := map[string]struct {
		statement string
		wantError string
	}{
		"the only row, with the default gap policy": {
			statement: `INSERT INTO settings.user_settings (id) VALUES (1)`,
		},
		"the only row, excluding gaps": {
			statement: `INSERT INTO settings.user_settings (id, gap_policy) VALUES (1, 'exclude')`,
		},
		"a second row": {
			statement: `INSERT INTO settings.user_settings (id) VALUES (2)`,
			wantError: "user_settings_single_row_check",
		},
		"the reserved cost basis policy": {
			statement: `INSERT INTO settings.user_settings (id, gap_policy) VALUES (1, 'cost_basis')`,
			wantError: "user_settings_gap_policy_check",
		},
		"an unknown gap policy": {
			statement: `INSERT INTO settings.user_settings (id, gap_policy) VALUES (1, 'interpolate')`,
			wantError: "user_settings_gap_policy_check",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.DB.ExecContext(s.T().Context(), tc.statement)
			if tc.wantError != "" {
				require.ErrorContains(s.T(), err, tc.wantError)
				return
			}
			require.NoError(s.T(), err)

			stored := &model.UserSettings{}
			temporaltesting.Retrieve(s.T(), s.DB, stored, model.UserSettingsID)
			require.Equal(s.T(), model.GapPolicyExclude, stored.GapPolicy)
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.UserSettings)(nil), model.UserSettingsID)
	}
}
