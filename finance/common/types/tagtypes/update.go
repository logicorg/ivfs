package tagtypes

// UpdateTagParams is the input to an update tag operation
type UpdateTagParams struct {
	// ID is the ID of the tag to update
	ID int64
	// Name is a pointer. Set it to a non-nil value to update the tag's name.
	Name *string
	// Description is a pointer. Set it to a non-nil value to update the tag's description.
	Description *string
}
