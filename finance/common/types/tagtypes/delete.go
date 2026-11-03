package tagtypes

// DeleteTagParams holds the parameters for deleting a tag.
// You must provide either the ID or the Name of the tag to delete.
// If both are provided, the ID will be used.
type DeleteTagParams struct {
	// ID is the ID of the Tag to delete. If set, it takes precedence over Name
	ID int64
	// Name is the name of the Tag to delete.
	Name string
}
