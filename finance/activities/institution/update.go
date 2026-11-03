package institution

import (
	"context"
	"strings"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

func (a *Institution) Update(ctx context.Context, params it.UpdateInstitutionParams) (*it.UpdateInstitutionResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.FindBy.ID <= 0 && params.FindBy.Name == "" {
		return nil, errors.ErrInvalidParameter{Description: "at least one of ID or Name must be provided to find the institution"}
	}

	retrieved, err := a.Retrieve(ctx, params.FindBy)
	if err != nil {
		return nil, err
	}

	if name := strings.TrimSpace(params.Name); name != "" {
		retrieved.Institution.Name = name
	}

	// TODO: Validate that the kind is one of the allowed values. This should be done in the model layer.
	if params.Kind != "" {
		retrieved.Institution.Kind = params.Kind
	}

	_, err = db.NewUpdate().Model(retrieved.Institution).WherePK().Exec(ctx)
	if conflict := sqllib.AlreadyExists(err, "Institution"); conflict != nil {
		return nil, conflict
	}
	if err != nil {
		return nil, err
	}

	return &it.UpdateInstitutionResult{ID: retrieved.Institution.ID}, nil
}
