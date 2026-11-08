package tag

import (
	"context"
	"fmt"
	"strings"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// InsertTag returns the tag with the given name, compared case-insensitively,
// creating it when there is none (ADR 0005). A tag that already exists is
// returned unchanged, whatever description the input carries. The name is
// stored trimmed, in the casing it was first given.
func (t *Tag) InsertTag(ctx context.Context, params tt.InsertTagParams) (*tt.InsertTagResult, error) {
	db, err := t.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"Name"}}
	}

	tag := &model.Tag{Name: name}
	if params.Description != nil {
		tag.Description = *params.Description
	}

	_, err = db.NewInsert().Model(tag).Returning("id").Exec(ctx)
	switch {
	case err == nil:
		return &tt.InsertTagResult{ID: tag.ID}, nil
	case sqllib.IsAlreadyExists(err):
		tag := &model.Tag{}
		err := model.FindTagByNameQuery(db, name).Scan(ctx, tag)
		if err == nil {
			return &tt.InsertTagResult{ID: tag.ID}, err
		}
		return nil, fmt.Errorf("detected conflict but failed to retrieve tag: %w", err)
	default:
		return nil, fmt.Errorf("failed to insert tag: %w", err)
	}
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
