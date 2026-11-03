package settings

import (
	"testing"

	"github.com/logicorg/ivfs/finance/activities/user/settings"
	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type UpdateWorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestUpdateWorkflowUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateWorkflowUnitTestSuite))
}

func (s *UpdateWorkflowUnitTestSuite) TestUpdateUserSettings() {
	eur := int64(1)

	tests := map[string]struct {
		params st.UpdateUserSettingsParams
		// updates is whether the activity is reached.
		updates   bool
		err       error
		wantError string
	}{
		"a default currency": {
			params:  st.UpdateUserSettingsParams{DefaultCurrencyID: &eur, GapPolicy: model.GapPolicyExclude},
			updates: true,
		},
		"the reserved cost basis policy is refused before the activity": {
			params:    st.UpdateUserSettingsParams{DefaultCurrencyID: &eur, GapPolicy: model.GapPolicyCostBasis},
			wantError: "the gap policy cost_basis is not available yet; use exclude",
		},
		"a currency that does not exist": {
			params:    st.UpdateUserSettingsParams{DefaultCurrencyID: &eur, GapPolicy: model.GapPolicyExclude},
			updates:   true,
			err:       errors.ErrNotFound{ResourceType: "Currency", ID: "1"},
			wantError: "resource Currency not found with ID: 1",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			sa := settings.Activities{}
			stored := &model.UserSettings{ID: model.UserSettingsID, DefaultCurrencyID: tc.params.DefaultCurrencyID, GapPolicy: tc.params.GapPolicy}
			if tc.updates {
				var result *st.UpdateUserSettingsResult
				if tc.err == nil {
					result = &st.UpdateUserSettingsResult{Settings: stored}
				}
				env.OnActivity(sa.UpdateUserSettings, mock.Anything, tc.params).Return(result, tc.err).Once()
			}

			env.ExecuteWorkflow(UpdateUserSettings, tc.params)

			env.AssertExpectations(s.T())
			if tc.wantError != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *st.UpdateUserSettingsResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), stored.DefaultCurrencyID, result.Settings.DefaultCurrencyID)
			assert.Equal(s.T(), stored.GapPolicy, result.Settings.GapPolicy)
		})
	}
}
