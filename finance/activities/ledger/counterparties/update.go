package counterparties

import (
	"context"
	"fmt"
	"strings"

	cpt "github.com/logicorg/ivfs/finance/common/types/counterpartytypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

func (a *Activities) RenameCounterparty(ctx context.Context, input cpt.RenameCounterpartyInput) (*cpt.CounterpartyMutationResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	input.Name = strings.TrimSpace(input.Name)

	result := &cpt.CounterpartyMutationResult{CounterpartyID: input.CounterpartyID}
	res, err := db.NewUpdate().
		Model(&model.Counterparty{ID: input.CounterpartyID, Name: input.Name}).
		Column("name", "updated_at").
		WherePK().
		Exec(ctx)

	if conflict := sqllib.AlreadyExists(err, "Counterparty"); conflict != nil {
		return nil, conflict
	}
	if err != nil {
		return nil, fmt.Errorf("rename counterparty: %w", err)
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return nil, internalerrors.ErrNotFound{ResourceType: "Counterparty", ID: input.CounterpartyID}
	}

	return result, nil
}
