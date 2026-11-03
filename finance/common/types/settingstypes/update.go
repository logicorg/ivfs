package settingstypes

import (
	"fmt"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// UpdateUserSettingsParams replaces the user's settings as a whole, as a PUT
// does: a nil DefaultCurrencyID clears the default currency.
type UpdateUserSettingsParams struct {
	DefaultCurrencyID *int64
	GapPolicy         model.GapPolicy
}

// Validate checks the gap policy is one that is built and the currency ID, if
// any, is positive.
func (p *UpdateUserSettingsParams) Validate() error {
	switch p.GapPolicy {
	case "":
		return errors.ErrMissingRequiredParameter{Parameters: []string{"GapPolicy"}}
	case model.GapPolicyExclude:
	case model.GapPolicyCostBasis:
		return errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the gap policy %s is not available yet; use %s", model.GapPolicyCostBasis, model.GapPolicyExclude)}
	default:
		return errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"%q is not a gap policy; use %s", p.GapPolicy, model.GapPolicyExclude)}
	}
	if p.DefaultCurrencyID != nil && *p.DefaultCurrencyID <= 0 {
		return errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the default currency ID must be positive, got %d", *p.DefaultCurrencyID)}
	}
	return nil
}

// UpdateUserSettingsResult holds the settings as stored.
type UpdateUserSettingsResult struct {
	Settings *model.UserSettings
}
