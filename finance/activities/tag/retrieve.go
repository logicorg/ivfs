package tag

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// RetrieveTag retrives a tag
func (t *Tag) RetrieveTag(ctx context.Context, params tt.RetrieveTagParams) (*tt.RetrieveTagResult, error) {
	db, err := t.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(params.Name)
	if params.ID == 0 && name == "" {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ID", "Name"}}
	}

	var query *bun.SelectQuery
	if params.ID != 0 {
		query = model.FindTagByIDQuery(db, params.ID)
	} else {
		query = model.FindTagByNameQuery(db, name)
	}

	tag := &model.Tag{}
	err = query.Scan(ctx, tag)
	switch err {
	case nil:
		return &tt.RetrieveTagResult{Tag: tag}, nil
	case sql.ErrNoRows:
		return nil, errors.ErrNotFound{
			ResourceType: "Tag",
			ID:           fmt.Sprintf("ID: %d, Name: '%s'", params.ID, name),
		}
	default:
		return nil, fmt.Errorf("failed to retrieve tag: %w", err)
	}
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
