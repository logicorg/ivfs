package tag

import (
	"context"
	"strings"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// DeleteTag deletes a tag
func (t *Tag) DeleteTag(ctx context.Context, params tt.DeleteTagParams) error {
	db, err := t.Pools.For(ctx)
	if err != nil {
		return err
	}

	if params.ID == 0 && params.Name == "" {
		return errors.ErrMissingRequiredParameter{Parameters: []string{"ID", "Name"}}
	}

	query := db.NewDelete().Model((*model.Tag)(nil))
	if params.ID != 0 {
		query.Where("id = ?", params.ID)
	} else {
		query.Where("lower(name) = lower(?)", strings.TrimSpace(params.Name))
	}
	_, err = query.Exec(ctx)

	return sqllib.IgnoreNotFound(err)
}
