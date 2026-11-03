package assets

import (
	"time"

	"github.com/logicorg/ivfs/web/api/gateway"
)

// Asset is an instrument as the API returns it, not a holding of it.
type Asset struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind" enum:"equity,stock,etf,fund,bond,crypto,commodity,other"`
	Ticker      string `json:"ticker,omitempty"`
	ISIN        string `json:"isin,omitempty" doc:"ISO 6166 identifier"`
	Exchange    string `json:"exchange,omitempty"`
	Description string `json:"description,omitempty"`
}

// AssetList is the assets that match, by name.
type AssetList struct {
	Assets []Asset `json:"assets"`
}

// ListAssetsInput is the request of GET /api/assets.
type ListAssetsInput struct {
	NameContains   string `query:"nameContains" doc:"only assets whose name contains this, in any case"`
	TickerContains string `query:"tickerContains" doc:"only assets whose ticker contains this, in any case"`
}

// ListAssetsOutput is the response of GET /api/assets.
type ListAssetsOutput struct {
	Body AssetList
}

// AssetForm is the answers to the form that creates an asset.
type AssetForm struct {
	Name        string `json:"name" minLength:"1" doc:"the asset's name; an asset with this name already is returned as it is"`
	Kind        string `json:"kind" enum:"equity,stock,etf,fund,bond,crypto,commodity,other"`
	Ticker      string `json:"ticker" required:"false" doc:"the symbol it trades under, such as VWCE"`
	Description string `json:"description" required:"false"`
}

// CreatedAsset is the id of the asset with the form's name, created now or
// before.
type CreatedAsset struct {
	ID int64 `json:"id"`
}

// CreateAssetInput is the request of POST /api/assets.
type CreateAssetInput struct {
	Body AssetForm
}

// CreateAssetOutput is the response of POST /api/assets.
type CreateAssetOutput struct {
	Body CreatedAsset
}

// ListAssetValuesInput is the request of GET /api/assets/{id}/values.
type ListAssetValuesInput struct {
	ID int64 `path:"id" minimum:"1"`
	gateway.RangeInput
}

// AssetValue is what one unit of an asset was worth on a date, according to
// a provider.
type AssetValue struct {
	Date       string         `json:"date" format:"date"`
	Value      gateway.Amount `json:"value"`
	CurrencyID int64          `json:"currencyId"`
	ObservedAt time.Time      `json:"observedAt" doc:"when the value was really observed: before date on a day that repeats an earlier one, such as a Saturday"`
}

// ProviderValues are one provider's values of the asset.
type ProviderValues struct {
	Provider string       `json:"provider"`
	Values   []AssetValue `json:"values" doc:"oldest first, from the first date the provider has a value for"`
}

// AssetValues are the asset's values from every provider that has some, and
// the providers that failed, each in the configured order.
type AssetValues struct {
	Values []ProviderValues        `json:"values"`
	Errors []gateway.ProviderError `json:"errors"`
}

// ListAssetValuesOutput is the response of GET /api/assets/{id}/values.
type ListAssetValuesOutput struct {
	Body AssetValues
}
