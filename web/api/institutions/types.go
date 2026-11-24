package institutions

// CloseInstitutionInput is the request of POST /api/institutions/{id}/close.
type CloseInstitutionInput struct {
	ID int64 `path:"id" minimum:"1"`
}

// DeletionPreviewInput is the request of GET
// /api/institutions/{id}/deletion-preview.
type DeletionPreviewInput struct {
	ID int64 `path:"id" minimum:"1"`
}

// DeleteInstitutionInput is the request of DELETE /api/institutions/{id}.
type DeleteInstitutionInput struct {
	ID int64 `path:"id" minimum:"1"`
}
