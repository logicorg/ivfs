package institution

import (
	"context"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

func (a *Institution) Retrieve(ctx context.Context, params it.RetrieveInstitutionParams) (*it.RetrieveInstitutionResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.ID <= 0 && params.Name == "" {
		return nil, errors.ErrInvalidParameter{Description: "either ID or Name must be provided"}
	}
	if params.ID > 0 && params.Name != "" {
		return nil, errors.ErrInvalidParameter{Description: "only one of ID or Name may be provided"}
	}

	retrieved := new(model.Institution)
	query := model.FindInstitutionByNameQuery(db, params.Name)
	if params.ID > 0 {
		query = model.FindInstitutionByIDQuery(db, params.ID)
	}
	err = query.Scan(ctx, retrieved)
	switch {
	case err == nil:
		return &it.RetrieveInstitutionResult{Institution: retrieved}, nil
	case sqllib.IsNotFound(err):
		return nil, errors.ErrNotFound{ResourceType: "Institution"}
	default:
		return nil, err
	}
}
