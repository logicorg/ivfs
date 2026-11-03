package tag

import (
	"context"
	"database/sql"
	"fmt"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// InsertTag inserts a tag
func (t *Tag) InsertTag(ctx context.Context, params tt.InsertTagParams) (*tt.InsertTagResult, error) {
	db, err := t.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.Name == "" {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"Name"}}
	}

	tag := &model.Tag{
		Name: params.Name,
	}
	if params.Description != nil {
		tag.Description = *params.Description
	}

	existing := &model.Tag{}
	err = model.FindTagByNameQuery(db, tag.Name).Scan(ctx, existing)
	switch err {
	case sql.ErrNoRows:
	case nil:
		return &tt.InsertTagResult{ID: existing.ID}, nil
	default:
		return nil, fmt.Errorf("failed to check for existing tag: %w", err)
	}

	_, err = db.NewInsert().Model(tag).Returning("id").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to insert tag: %w", err)
	}
	return &tt.InsertTagResult{ID: tag.ID}, nil
}

// InsertTags inserts multiple tags
func (t *Tag) InsertTags(ctx context.Context, params []string) ([]*tt.InsertTagResult, error) {
	if len(params) == 0 {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"params"}}
	}

	results := make([]*tt.InsertTagResult, 0, len(params))
	for _, name := range params {
		res, err := t.InsertTag(ctx, tt.InsertTagParams{Name: name})
		if err != nil {
			return nil, fmt.Errorf("failed to insert tag '%s': %w", name, err)
		}
		results = append(results, res)
	}

	return results, nil
}
