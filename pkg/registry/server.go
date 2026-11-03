package registry

import (
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// Server is the PostgreSQL server that holds the registry and every user's
// database. One DSN reaches all of them, so the role it signs in as is the
// one that owns the user databases.
type Server struct {
	DSN string
}

// Open returns a pool on the named database of the server. The database in
// the DSN is only the default: the name given here replaces it.
func (s Server) Open(database string) *bun.DB {
	connector := pgdriver.NewConnector(pgdriver.WithDSN(s.DSN), pgdriver.WithDatabase(database))
	return bun.NewDB(sql.OpenDB(connector), pgdialect.New())
}
