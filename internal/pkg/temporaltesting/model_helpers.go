package temporaltesting

import (
	"fmt"
	"testing"

	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// Insert inserts an entry of model M in the database
func Insert[M any](t *testing.T, db *bun.DB, m *M) {
	t.Helper()

	_, err := db.NewInsert().Model(m).Returning("id").Exec(t.Context())
	require.NoError(t, err)
}

func InsertOrRetrieve[M any](t *testing.T, db *bun.DB, m *M, conflictColumn string) {
	t.Helper()

	conflictPolicy := fmt.Sprintf("CONFLICT (%s) DO UPDATE", conflictColumn)

	_, err := db.NewInsert().Model(m).On(conflictPolicy).Returning("id").Exec(t.Context())
	require.NoError(t, err)
}

func Retrieve[M any](t *testing.T, db *bun.DB, m *M, id int64) {
	t.Helper()

	err := db.NewSelect().Model(m).Where("id = ?", id).Scan(t.Context())
	require.NoError(t, err)
}

// Delete deletes an entry from the database
func Delete[M any](t *testing.T, db *bun.DB, model M, id int64) {
	t.Helper()

	_, err := db.NewDelete().Model(model).Where("id = ?", id).Exec(t.Context())
	require.NoError(t, err)
}

// DeleteWhere works like `Delete` but let you specify the template where clause.
// You can pass template `my_field` and value `1` to have a WHERE clause added to the
// delete query like `WHERE my_field = 1`.
func DeleteWhere[M any](t *testing.T, db *bun.DB, model M, template string, value any) {
	t.Helper()

	addWhere := func(q bun.QueryBuilder) bun.QueryBuilder {
		return q.Where(template, value)
	}

	q := db.NewDelete().Model(model)

	q.ApplyQueryBuilder(addWhere)

	_, err := q.Exec(t.Context())
	require.NoError(t, err)
}

// DeleteObj deletes an entry from the database using the primary key of the model.
func DeleteObj[M any](t *testing.T, db *bun.DB, model M) {
	t.Helper()

	_, err := db.NewDelete().Model(model).WherePK().Exec(t.Context())
	require.NoError(t, err)
}

// DeleteIgnoreNotFound is like Delete, but does not fail on not found errors
func DeleteIgnoreNotFound[M any](t *testing.T, db *bun.DB, model M, id int64) {
	t.Helper()

	_, err := db.NewDelete().Model(model).Where("id = ?", id).Exec(t.Context())
	require.NoError(t, sqllib.IgnoreNotFound(err))
}
