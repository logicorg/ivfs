package registry_test

import (
	"github.com/google/uuid"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/require"
)

// Test_Init checks that running Init on a registry that has users keeps them,
// since every provisioning worker runs it at start.
func (s *UnitTestSuite) Test_Init() {
	user := &registry.User{ID: uuid.Must(uuid.NewV7()), Issuer: "https://idp.test", Subject: "alice"}
	temporaltesting.Insert(s.T(), s.DB, user)

	require.NoError(s.T(), registry.Init(s.T().Context(), s.DB))

	got := &registry.User{ID: user.ID}
	require.NoError(s.T(), s.DB.NewSelect().Model(got).WherePK().Scan(s.T().Context()))
	require.Equal(s.T(), user.Subject, got.Subject)
}

// Test_Init_Identity checks that an identity is unique as an exact (issuer,
// subject) pair, and nothing looser: the same subject at another issuer, or in
// another case, is someone else.
func (s *UnitTestSuite) Test_Init_Identity() {
	tcs := map[string]struct {
		issuer, subject string
		wantConflict    bool
	}{
		"the same identity":              {issuer: "https://idp.test", subject: "alice", wantConflict: true},
		"the same subject, other issuer": {issuer: "https://other.test", subject: "alice"},
		"the subject in another case":    {issuer: "https://idp.test", subject: "Alice"},
	}

	for name, tc := range tcs {
		s.Run(name, func() {
			first := &registry.User{ID: uuid.Must(uuid.NewV7()), Issuer: "https://idp.test", Subject: "alice"}
			temporaltesting.Insert(s.T(), s.DB, first)
			defer temporaltesting.DeleteObj(s.T(), s.DB, first)

			second := &registry.User{ID: uuid.Must(uuid.NewV7()), Issuer: tc.issuer, Subject: tc.subject}
			_, err := s.DB.NewInsert().Model(second).Exec(s.T().Context())
			if !tc.wantConflict {
				require.NoError(s.T(), err)
				temporaltesting.DeleteObj(s.T(), s.DB, second)
				return
			}
			var conflict internalerrors.ErrResourceConflict
			require.ErrorAs(s.T(), sqllib.AlreadyExists(err, "User"), &conflict)
		})
	}
}
