package tagtypes

// InsertTagParams is the input to an insert tag operation
type InsertTagParams struct {
	Name        string
	Description *string
}

// InsertTagResult is the result of an insert tag operation
type InsertTagResult struct {
	// ID is the unique identifier for the newly created tag.
	ID int64
}
