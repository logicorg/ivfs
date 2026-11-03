package institution

import (
	"context"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// Delete deletes an institution by ID or Name. Deleting an institution that
// does not exist is not an error, so retries remain idempotent. An institution
// that financial accounts still refer to cannot be deleted, and the caller gets
// an ErrResourceConflict; it can be closed instead.
func (a *Institution) Delete(ctx context.Context, params it.DeleteInstitutionParams) (it.DeleteInstitutionResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return it.DeleteInstitutionResult{}, err
	}

	if err := params.Validate(); err != nil {
		return it.DeleteInstitutionResult{}, err
	}

	query := db.NewDelete().Model((*model.Institution)(nil))
	if params.FindBy.ID > 0 {
		query = query.Where("id = ?", params.FindBy.ID)
	} else {
		query = query.Where("name = ?", params.FindBy.Name)
	}
	_, err = query.Exec(ctx)
	if conflict := sqllib.StillReferenced(err, "Institution"); conflict != nil {
		return it.DeleteInstitutionResult{}, conflict
	}
	return it.DeleteInstitutionResult{}, sqllib.IgnoreNotFound(err)
}
