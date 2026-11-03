package settings

import (
	"testing"

	"github.com/logicorg/ivfs/finance/activities/user/settings"
	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type RetrieveWorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestRetrieveWorkflowUnitTestSuite(t *testing.T) {
	suite.Run(t, new(RetrieveWorkflowUnitTestSuite))
}

func (s *RetrieveWorkflowUnitTestSuite) TestRetrieveUserSettings() {
	eur := int64(1)
	stored := &model.UserSettings{ID: model.UserSettingsID, DefaultCurrencyID: &eur, GapPolicy: model.GapPolicyExclude}

	tests := map[string]struct {
		result    *st.RetrieveUserSettingsResult
		err       error
		wantError string
	}{
		"the settings": {
			result: &st.RetrieveUserSettingsResult{Settings: stored},
		},
		"the activity fails": {
			err:       assert.AnError,
			wantError: assert.AnError.Error(),
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			sa := settings.Activities{}
			env.OnActivity(sa.RetrieveUserSettings, mock.Anything).Return(tc.result, tc.err)

			env.ExecuteWorkflow(RetrieveUserSettings, st.RetrieveUserSettingsParams{})

			if tc.wantError != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *st.RetrieveUserSettingsResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result.Settings.DefaultCurrencyID, result.Settings.DefaultCurrencyID)
			assert.Equal(s.T(), tc.result.Settings.GapPolicy, result.Settings.GapPolicy)
		})
	}
}
