package sqllib_test

import (
	"fmt"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/require"
)

// Test_AlreadyExists checks that a unique violation is recognised through the
// wrapping an activity adds, and reports the key that is taken.
func (s *UnitTestSuite) Test_AlreadyExists() {
	temporaltesting.Insert(s.T(), s.DB, testhelpers.Euro())
	_, err := s.DB.NewInsert().Model(testhelpers.Euro()).Exec(s.T().Context())
	require.Error(s.T(), err)

	mapped := sqllib.AlreadyExists(fmt.Errorf("insert currency: %w", err), "Currency")

	var conflict internalerrors.ErrResourceConflict
	require.ErrorAs(s.T(), mapped, &conflict)
	require.Equal(s.T(), "Currency", conflict.ResourceType)
	require.Contains(s.T(), conflict.Reason, "already exists")
}

// Test_AlreadyExists_OtherErrorsAreNotMapped checks that errors that are not a
// unique violation pass through untouched, so that callers keep handling them
// as before.
func (s *UnitTestSuite) Test_AlreadyExists_OtherErrorsAreNotMapped() {
	_, fkErr := s.DB.NewInsert().Model(&model.FinancialAccount{InstitutionID: 999999, Name: "orphan"}).Exec(s.T().Context())
	require.Error(s.T(), fkErr)

	tests := map[string]error{
		"a foreign key violation": fkErr,
		"no error":                nil,
	}

	for name, err := range tests {
		s.Run(name, func() {
			require.NoError(s.T(), sqllib.AlreadyExists(err, "Currency"))
		})
	}
}
