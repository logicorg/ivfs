package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	"github.com/logicorg/ivfs/pkg/model"
)

// RetrieveUserSettings returns the user's settings. A user who has never
// changed them has no row, and gets the defaults rather than ErrNotFound: the
// settings always exist, even before they are stored.
func (a *Activities) RetrieveUserSettings(ctx context.Context) (*st.RetrieveUserSettingsResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	settings := &model.UserSettings{}
	err = db.NewSelect().Model(settings).Where("us.id = ?", model.UserSettingsID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return &st.RetrieveUserSettingsResult{Settings: model.DefaultUserSettings()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("retrieve user settings: %w", err)
	}
	return &st.RetrieveUserSettingsResult{Settings: settings}, nil
}
