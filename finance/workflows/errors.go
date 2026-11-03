package workflows

import (
	"errors"

	"go.temporal.io/sdk/temporal"
)

// HasErrorType reports whether any temporal.ApplicationError in err's chain
// has the given type, such as "ErrNotFound". A single errors.As is not enough
// across a child workflow: the child wraps its activity's structured error
// with fmt.Errorf, so the first ApplicationError in the chain has the type
// "wrapError", and the structured one is its cause (see docs/gotchas.md).
func HasErrorType(err error, errType string) bool {
	_, ok := FindErrorType(err, errType)
	return ok
}

// FindErrorType returns the first temporal.ApplicationError in err's chain
// with the given type, searched the way HasErrorType does, so a caller can
// read its message.
func FindErrorType(err error, errType string) (*temporal.ApplicationError, bool) {
	for err != nil {
		var applicationErr *temporal.ApplicationError
		if !errors.As(err, &applicationErr) {
			return nil, false
		}
		if applicationErr.Type() == errType {
			return applicationErr, true
		}
		err = applicationErr.Unwrap()
	}
	return nil, false
}
