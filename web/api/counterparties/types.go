package counterparties

// Counterparty is who money is paid to or received from, as the API returns
// it.
type Counterparty struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	ExternalID *string `json:"externalId,omitempty" doc:"how an external source, such as a bank export, identifies it"`
}

// CounterpartyList is every counterparty asked for, sorted by name.
type CounterpartyList struct {
	Counterparties []Counterparty `json:"counterparties"`
}

// ListCounterpartiesInput is the request of GET /api/counterparties.
type ListCounterpartiesInput struct {
	NameContains string `query:"nameContains" doc:"only counterparties whose name contains this, ignoring case; every one if omitted"`
}

// ListCounterpartiesOutput is the response of GET /api/counterparties.
type ListCounterpartiesOutput struct {
	Body CounterpartyList
}

// NewCounterparty is a counterparty to create.
type NewCounterparty struct {
	Name       string  `json:"name" minLength:"1"`
	ExternalID *string `json:"externalId,omitempty" minLength:"1" doc:"how an external source, such as a bank export, identifies it"`
}

// CreateCounterpartyInput is the request of POST /api/counterparties.
type CreateCounterpartyInput struct {
	Body NewCounterparty
}

// CreatedCounterparty is the id of a counterparty, created now or before.
type CreatedCounterparty struct {
	ID int64 `json:"id"`
}

// CreateCounterpartyOutput is the response of POST /api/counterparties.
type CreateCounterpartyOutput struct {
	Body CreatedCounterparty
}

// CounterpartyName is a counterparty's new name.
type CounterpartyName struct {
	Name string `json:"name" minLength:"1"`
}

// RenameCounterpartyInput is the request of
// POST /api/counterparties/{id}/rename.
type RenameCounterpartyInput struct {
	ID   int64 `path:"id" minimum:"1"`
	Body CounterpartyName
}
