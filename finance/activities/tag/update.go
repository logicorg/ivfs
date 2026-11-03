package tag

import (
	"context"
	"database/sql"
	"fmt"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// UpdateTag updates a tag
func (t *Tag) UpdateTag(ctx context.Context, param tt.UpdateTagParams) error {
	db, err := t.Pools.For(ctx)
	if err != nil {
		return err
	}

	var tag model.Tag

	if err := model.FindTagByIDQuery(db, param.ID).Scan(ctx, &tag); err != nil {
		if err == sql.ErrNoRows {
			return errors.ErrNotFound{ResourceType: "Tag", ID: param.ID}
		}
		return fmt.Errorf("failed to retrieve tag: %w", err)
	}

	if param.Name != nil {
		tag.Name = *param.Name
	}
	if param.Description != nil {
		tag.Description = *param.Description
	}

	_, err = db.NewUpdate().Model(&tag).Where("id = ?", param.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}

	return nil
}
