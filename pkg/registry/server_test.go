package registry_test

import (
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/stretchr/testify/require"
)

// Test_Server_Open checks that the name given to Open, not the database in
// the DSN, is the one the pool connects to.
func (s *UnitTestSuite) Test_Server_Open() {
	dsn, err := s.Postgres().ConnectionString(s.T().Context(), "sslmode=disable")
	require.NoError(s.T(), err)
	server := registry.Server{DSN: dsn}

	tcs := map[string]struct {
		database string
	}{
		"the database in the DSN": {database: "database"},
		"another database":        {database: "postgres"},
	}

	for name, tc := range tcs {
		s.Run(name, func() {
			db := server.Open(tc.database)
			defer db.Close()

			var got string
			require.NoError(s.T(), db.QueryRowContext(s.T().Context(), "SELECT current_database()").Scan(&got))
			require.Equal(s.T(), tc.database, got)
		})
	}
}
