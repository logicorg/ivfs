package institutiontypes

// DeleteInstitutionParams selects the institution to delete, by ID or Name.
type DeleteInstitutionParams struct {
	FindBy RetrieveInstitutionParams
}

// Validate checks if the provided parameters form a valid request.
func (i *DeleteInstitutionParams) Validate() error {
	return i.FindBy.Validate()
}

// DeleteInstitutionResult holds the result of a delete institution operation.
type DeleteInstitutionResult struct{}
