package model

import (
	"time"

	"github.com/uptrace/bun"
)

// GapPolicy is what net worth does on a date an asset has no value, or a
// currency no rate, for (ADR 0015).
type GapPolicy string

const (
	// GapPolicyExclude leaves out what cannot be valued, and flags it.
	GapPolicyExclude GapPolicy = "exclude"
	// GapPolicyCostBasis counts a holding without a value at its cost basis.
	// It is reserved: the database rejects it until it is built.
	GapPolicyCostBasis GapPolicy = "cost_basis"
)

// UserSettingsID is the id of the only settings row. Each user has a database
// of their own (ADR 0013), so their settings need no user column.
const UserSettingsID int64 = 1

// UserSettings are a user's preferences, such as the currency net worth is
// shown in.
type UserSettings struct {
	bun.BaseModel `bun:"table:settings.user_settings,alias:us"`

	ID int64 `bun:"id,pk" json:"id"`
	// DefaultCurrencyID is the currency net worth is shown in when the
	// dashboard is not given one, or nil when the user has not chosen one.
	DefaultCurrencyID *int64 `bun:"default_currency_id" json:"defaultCurrencyId,omitempty"`
	// GapPolicy is what net worth does on a date without a value or a rate.
	GapPolicy GapPolicy `bun:"gap_policy,notnull,default:'exclude'" json:"gapPolicy"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	DefaultCurrency *Currency `bun:"rel:belongs-to,join:default_currency_id=id,on_delete:restrict" json:"defaultCurrency,omitempty"`
}

// DefaultUserSettings are the settings of a user who has not changed any.
func DefaultUserSettings() *UserSettings {
	return &UserSettings{ID: UserSettingsID, GapPolicy: GapPolicyExclude}
}
