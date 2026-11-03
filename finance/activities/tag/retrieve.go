package tag

import (
	"context"
	"database/sql"
	"fmt"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// RetrieveTag retrives a tag
func (t *Tag) RetrieveTag(ctx context.Context, params tt.RetrieveTagParams) (*tt.RetrieveTagResult, error) {
	db, err := t.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.ID == 0 && params.Name == "" {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ID", "Name"}}
	}

	query := db.NewSelect().Model((*model.Tag)(nil))
	if params.ID != 0 {
		query.Where("id = ?", params.ID)
	} else {
		query.Where("name = ?", params.Name)
	}

	tag := &model.Tag{}
	if err := query.Scan(ctx, tag); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound{
				ResourceType: "Tag",
				ID:           fmt.Sprintf("ID: %d, Name: '%s'", params.ID, params.Name),
			}
		}
		return nil, fmt.Errorf("failed to retrieve tag: %w", err)
	}

	return &tt.RetrieveTagResult{Tag: tag}, nil
}

// ListTags list the tags available in the DB
func (t *Tag) ListTags(ctx context.Context) ([]*model.Tag, error) {
	db, err := t.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	tags := make([]*model.Tag, 0)
	err = db.NewSelect().Model(&tags).Order("name ASC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, nil
}
