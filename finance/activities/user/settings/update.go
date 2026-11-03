package settings

import (
	"context"
	"fmt"

	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// UpdateUserSettings replaces the user's settings. The row is created on the
// first update, so the update is an upsert, and a retry stores the same row.
func (a *Activities) UpdateUserSettings(ctx context.Context, params st.UpdateUserSettingsParams) (*st.UpdateUserSettingsResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	settings := &model.UserSettings{
		ID:                model.UserSettingsID,
		DefaultCurrencyID: params.DefaultCurrencyID,
		GapPolicy:         params.GapPolicy,
	}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(settings).
			On("CONFLICT (id) DO UPDATE").
			Set("default_currency_id = EXCLUDED.default_currency_id").
			Set("gap_policy = EXCLUDED.gap_policy").
			Set("updated_at = current_timestamp").
			Returning("*").
			Exec(ctx)
		if err != nil {
			if notFound := sqllib.ParentNotFound(err); notFound != nil {
				return notFound
			}
			return fmt.Errorf("upsert user settings: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &st.UpdateUserSettingsResult{Settings: settings}, nil
}
