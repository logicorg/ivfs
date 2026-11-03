package errors

import (
	"fmt"
	"strings"
)

// ErrMissingRequiredParameter is an error type that indicates that one or more required parameters are missing.
// It contains an (optional) slice of parameter names that are missing.
type ErrMissingRequiredParameter struct {
	Parameters []string
}

func (e ErrMissingRequiredParameter) Error() string {
	if len(e.Parameters) == 0 {
		return "missing required parameters"
	}
	return "missing required parameters: " + strings.Join(e.Parameters, ", ")
}

// ErrInvalidParameter is an error caused by invalid parameters
type ErrInvalidParameter struct {
	Description string
}

func (e ErrInvalidParameter) Error() string {
	if e.Description == "" {
		return "invalid parameter"
	}
	return "invalid parameter: " + e.Description
}

// ErrNotFound is a not found error
type ErrNotFound struct {
	ResourceType string
	ID           interface{}

	Err error
}

func (e ErrNotFound) Error() string {
	if e.ResourceType == "" {
		return "resource not found"
	}
	if e.Err == nil {
		return fmt.Sprintf("resource %s not found with ID: %v", e.ResourceType, e.ID)
	}
	return fmt.Sprintf("resource %s not found with ID: %v: %v", e.ResourceType, e.ID, e.Err)

}

// ErrResourceConflict is an error caused by resource conflict
type ErrResourceConflict struct {
	ResourceType string
	Reason       string
}

func (e ErrResourceConflict) Error() string {
	if e.ResourceType == "" {
		return "resource conflict"
	}
	if e.Reason == "" {
		return fmt.Sprintf("resource %s conflict", e.ResourceType)
	}
	return fmt.Sprintf("resource %s conflict: %s", e.ResourceType, e.Reason)
}

// ErrResourceClosed indicates that an operation could not be completed
// because its target resource is closed, archived, or otherwise inactive
// (for example a closed Institution or FinancialAccount, or an archived
// Counterparty). It is a nonretryable precondition failure: retrying the
// same operation will not succeed until the resource's state changes, so
// callers should register its type name in a workflow's
// NonRetryableErrorTypes rather than retrying.
type ErrResourceClosed struct {
	ResourceType string
	ID           interface{}

	// Reason optionally overrides the generated message, e.g. "counterparty is archived".
	Reason string
}

func (e ErrResourceClosed) Error() string {
	if e.Reason != "" {
		return e.Reason
	}
	if e.ResourceType == "" {
		return "resource is closed"
	}
	if e.ID == nil {
		return fmt.Sprintf("resource %s is closed", e.ResourceType)
	}
	return fmt.Sprintf("resource %s with ID: %v is closed", e.ResourceType, e.ID)
}

// ValidationError is an error type that indicates that a validation error occurred.
type ValidationError struct {
	// Err is the underlying error.
	Err error
}

func (e ValidationError) Error() string {
	if e.Err == nil {
		return "validation error"
	}
	return fmt.Sprintf("validation error: %s", e.Err.Error())
}
