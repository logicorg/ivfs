package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/logicorg/ivfs/provisioning/common/types/usertypes"
	"github.com/uptrace/bun"
)

// InsertUser returns the user an identity belongs to, inserting it first when
// there is none. An existing user is returned unchanged, so a retry or a
// concurrent login gets the same uid.
func (a *Activities) InsertUser(ctx context.Context, params usertypes.InsertUserParams) (*usertypes.InsertUserResult, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate user id: %w", err)
	}

	user := &registry.User{ID: id, Issuer: params.Issuer, Subject: params.Subject}
	err = a.Registry.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// A concurrent insert of the same identity makes this one wait for it
		// to commit and then do nothing, so the select below, which takes a
		// fresh snapshot, always finds the row. An id generated here and not
		// used is simply never handed out.
		_, err := tx.NewInsert().Model(user).On("CONFLICT (issuer, subject) DO NOTHING").Exec(ctx)
		if err != nil {
			return fmt.Errorf("insert user: %w", err)
		}
		err = tx.NewSelect().Model(user).
			Where("u.issuer = ?", params.Issuer).
			Where("u.subject = ?", params.Subject).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("retrieve user: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &usertypes.InsertUserResult{ID: user.ID, Provisioned: user.ProvisionedAt != nil}, nil
}
