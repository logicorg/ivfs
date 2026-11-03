package settingstypes

import (
	"testing"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestUpdateUserSettingsParams_Validate(t *testing.T) {
	tests := map[string]struct {
		params    UpdateUserSettingsParams
		wantError string
	}{
		"a default currency": {
			params: UpdateUserSettingsParams{DefaultCurrencyID: new(int64(1)), GapPolicy: model.GapPolicyExclude},
		},
		"no default currency": {
			params: UpdateUserSettingsParams{GapPolicy: model.GapPolicyExclude},
		},
		"no gap policy": {
			params:    UpdateUserSettingsParams{DefaultCurrencyID: new(int64(1))},
			wantError: "missing required parameters: GapPolicy",
		},
		"the reserved cost basis policy": {
			params:    UpdateUserSettingsParams{GapPolicy: model.GapPolicyCostBasis},
			wantError: "the gap policy cost_basis is not available yet; use exclude",
		},
		"an unknown gap policy": {
			params:    UpdateUserSettingsParams{GapPolicy: "interpolate"},
			wantError: `"interpolate" is not a gap policy; use exclude`,
		},
		"a zero currency ID": {
			params:    UpdateUserSettingsParams{DefaultCurrencyID: new(int64(0)), GapPolicy: model.GapPolicyExclude},
			wantError: "the default currency ID must be positive, got 0",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}
