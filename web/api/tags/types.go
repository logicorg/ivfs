package tags

// Tag is a tag as the API returns it.
type Tag struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TagList is every tag, sorted by name.
type TagList struct {
	Tags []Tag `json:"tags"`
}

// ListTagsInput is the request of GET /api/tags.
type ListTagsInput struct{}

// ListTagsOutput is the response of GET /api/tags.
type ListTagsOutput struct {
	Body TagList
}

// NewTag is a tag to create.
type NewTag struct {
	Name        string  `json:"name" minLength:"1"`
	Description *string `json:"description,omitempty"`
}

// CreateTagInput is the request of POST /api/tags.
type CreateTagInput struct {
	Body NewTag
}

// CreatedTag is the id of a tag, created now or before.
type CreatedTag struct {
	ID int64 `json:"id"`
}

// CreateTagOutput is the response of POST /api/tags.
type CreateTagOutput struct {
	Body CreatedTag
}

// TagChanges are the fields of a tag to change. A field left out stays as it
// is.
type TagChanges struct {
	Name        *string `json:"name,omitempty" minLength:"1"`
	Description *string `json:"description,omitempty" doc:"empty clears it"`
}

// UpdateTagInput is the request of PATCH /api/tags/{id}.
type UpdateTagInput struct {
	ID   int64 `path:"id" minimum:"1"`
	Body TagChanges
}

// DeleteTagInput is the request of DELETE /api/tags/{id}.
type DeleteTagInput struct {
	ID int64 `path:"id" minimum:"1"`
}
