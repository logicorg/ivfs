package institution

import (
	"context"
	"fmt"
	"strings"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// Insert creates an institution. It is retrieve-or-insert on the name, compared
// case-insensitively (ADR 0005): if an institution with the same name already
// exists, its data is returned unchanged rather than creating a duplicate, so
// retrying is safe. The name is stored trimmed, in the casing it was first
// given.
func (a *Institution) Insert(ctx context.Context, param it.InsertInstitutionParams) (*it.InsertInstitutionResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := param.Validate(); err != nil {
		return nil, err
	}
	param.Name = strings.TrimSpace(param.Name)

	existing, err := a.findByName(ctx, param.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &it.InsertInstitutionResult{ID: existing.ID}, nil
	}

	institution := param.ToModel()
	_, err = db.NewInsert().Model(institution).Returning("id").Exec(ctx)
	if conflict := sqllib.AlreadyExists(err, "Institution"); conflict != nil {
		// Another insert wrote the same name after the lookup above: the unique
		// index held this one until it committed. Return its row.
		existing, err := a.findByName(ctx, param.Name)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, conflict
		}
		return &it.InsertInstitutionResult{ID: existing.ID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("insert institution: %w", err)
	}
	return &it.InsertInstitutionResult{ID: institution.ID}, nil
}

// findByName returns the institution with the given name, compared
// case-insensitively, or nil when there is none.
func (a *Institution) findByName(ctx context.Context, name string) (*model.Institution, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	existing := new(model.Institution)
	err = model.FindInstitutionByNameQuery(db, name).Scan(ctx, existing)
	switch {
	case sqllib.IsNotFound(err):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("check for existing institution: %w", err)
	default:
		return existing, nil
	}
}
