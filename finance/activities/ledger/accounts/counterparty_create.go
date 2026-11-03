package accounts

import (
	"context"
	"fmt"
	"strings"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// CreateCounterparty returns the counterparty with the given name, compared
// case-insensitively, creating it when there is none (ADR 0005). A counterparty
// that already exists is returned unchanged, whatever the rest of the input
// says, so a retry or a second create of the same merchant is harmless. The
// name is stored trimmed, in the casing it was first given.
func (a *Activities) CreateCounterparty(ctx context.Context, input lt.CreateCounterpartyInput) (*lt.CreateCounterpartyResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, internalerrors.ErrMissingRequiredParameter{Parameters: []string{"Name"}}
	}

	existing, err := findCounterpartyByName(ctx, db, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &lt.CreateCounterpartyResult{CounterpartyID: existing.ID}, nil
	}

	counterparty := &model.Counterparty{Name: name, ExternalID: input.ExternalID}
	_, err = db.NewInsert().Model(counterparty).Returning("id").Exec(ctx)
	if conflict := sqllib.AlreadyExists(err, "Counterparty"); conflict != nil {
		// Another create inserted the same name after the lookup above: the
		// unique index held this insert until it committed. Return its row.
		existing, err := findCounterpartyByName(ctx, db, name)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, conflict
		}
		return &lt.CreateCounterpartyResult{CounterpartyID: existing.ID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("insert counterparty: %w", err)
	}
	return &lt.CreateCounterpartyResult{CounterpartyID: counterparty.ID}, nil
}
