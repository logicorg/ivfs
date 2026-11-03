// Package sqllib contains utility function when dealing with a SQL database
package sqllib

import (
	"database/sql"
	"errors"
)

// IgnoreNotFound return nil if the error is a not found error (sql.ErrNoRows),
// otherwise returns the original error.
// This is useful to ignore "not found" errors in database operations.
func IgnoreNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

// IsNotFound checks if the error is a "not found" error (sql.ErrNoRows).
func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
