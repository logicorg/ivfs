package model

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type AssetKind string

const (
	AssetKindEquity    AssetKind = "equity"
	AssetKindStock     AssetKind = "stock"
	AssetKindETF       AssetKind = "etf"
	AssetKindFund      AssetKind = "fund"
	AssetKindBond      AssetKind = "bond"
	AssetKindCrypto    AssetKind = "crypto"
	AssetKindCommodity AssetKind = "commodity"
	AssetKindOther     AssetKind = "other"
)

// Asset identifies an instrument, not a particular holding of that
// instrument. Holdings are derived from PositionMovement records per financial account.
type Asset struct {
	bun.BaseModel `bun:"table:asset.asset,alias:as"`

	ID          int64     `bun:"id,pk,autoincrement" json:"id"`
	Name        string    `bun:"name,notnull" json:"name"`
	Kind        AssetKind `bun:"kind,notnull" json:"kind"`
	ISIN        *string   `bun:"isin,unique" json:"isin,omitempty" jsonschema:"ISO 6166 identifier"`
	Ticker      *string   `bun:"ticker" json:"ticker,omitempty"`
	Exchange    *string   `bun:"exchange" json:"exchange,omitempty"`
	Description *string   `bun:"description" json:"description,omitempty"`
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt   time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	Transactions []PositionMovement `bun:"rel:has-many,join:id=asset_id" json:"positionMovements,omitempty"`
	Values       []AssetValue       `bun:"rel:has-many,join:id=asset_id" json:"values,omitempty"`
}

// AssetValue is a market price used for net-worth valuation. It is not the
// Historical cost of a holding belongs to PositionMovement.
type AssetValue struct {
	bun.BaseModel `bun:"table:asset.asset_value,alias:av"`

	ID         int64           `bun:"id,pk,autoincrement" json:"id"`
	AssetID    int64           `bun:"asset_id,notnull" json:"assetId"`
	CurrencyID int64           `bun:"currency_id,notnull" json:"currencyId"`
	Value      decimal.Decimal `bun:"value,type:numeric(28,8),notnull" json:"value"`
	ObservedAt time.Time       `bun:"observed_at,nullzero,notnull" json:"observedAt"`
	Source     *string         `bun:"source" json:"source,omitempty"`

	Asset    *Asset    `bun:"rel:belongs-to,join:asset_id=id,on_delete:restrict" json:"asset,omitempty"`
	Currency *Currency `bun:"rel:belongs-to,join:currency_id=id,on_delete:restrict" json:"currency,omitempty"`
}

// FindAssetByNameQuery returns the select query to retrieve an asset by Name
func FindAssetByNameQuery(db *bun.DB, name string) *bun.SelectQuery {
	return db.NewSelect().Model((*Asset)(nil)).Where("name = ?", name)
}

// FindAssetByTickerQuery returns the select query to retrieve an asset by Ticker
func FindAssetByTickerQuery(db *bun.DB, ticker string) *bun.SelectQuery {
	return db.NewSelect().Model((*Asset)(nil)).Where("ticker = ?", ticker)
}

func FindAssetByIDQuery(db *bun.DB, id int64) *bun.SelectQuery {
	return db.NewSelect().Model((*Asset)(nil)).Where("id = ?", id)
}

func FindAssetByISINQuery(db *bun.DB, isin string) *bun.SelectQuery {
	return db.NewSelect().Model((*Asset)(nil)).Where("isin = ?", isin)
}
