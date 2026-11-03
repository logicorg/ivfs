package holdings

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// CostBasis is a cost in one currency.
type CostBasis struct {
	CurrencyID int64          `json:"currencyId"`
	Amount     gateway.Amount `json:"amount"`
}

// Holding is the units of one asset held in one financial account, and what
// they cost, replayed at average cost from the movements (ADR 0009).
type Holding struct {
	FinancialAccountID int64          `json:"financialAccountId"`
	AssetID            int64          `json:"assetId"`
	Units              gateway.Amount `json:"units"`
	// CostBasis is in each currency the units were paid in, which are not
	// converted, so they cannot be added together.
	CostBasis        []CostBasis `json:"costBasis" doc:"the cost of the units held, in each currency it was paid in"`
	UnknownCostBasis bool        `json:"unknownCostBasis" doc:"some units came without a cost, so costBasis is only part of it"`
	AverageCost      *CostBasis  `json:"averageCost,omitempty" doc:"the cost of one unit; absent with no units, an unknown basis, or several currencies"`
}

// HoldingList is the holdings that match.
type HoldingList struct {
	Holdings []Holding `json:"holdings"`
}

// ListHoldingsInput is the request of GET /api/holdings.
type ListHoldingsInput struct {
	FinancialAccountID int64     `query:"financialAccountId" minimum:"1" doc:"only the holdings of this financial account"`
	AssetID            int64     `query:"assetId" minimum:"1" doc:"only the holdings of this asset"`
	At                 time.Time `query:"at" doc:"count only movements that occurred at or before this instant; now if omitted"`
}

// ListHoldingsOutput is the response of GET /api/holdings.
type ListHoldingsOutput struct {
	Body HoldingList
}

// Movement is a change to the units of an asset held in a financial account.
type Movement struct {
	ID                 int64          `json:"id"`
	Kind               string         `json:"kind" enum:"trade,vest,split,transfer,adjustment"`
	FinancialAccountID int64          `json:"financialAccountId"`
	AssetID            int64          `json:"assetId"`
	UnitsDelta         gateway.Amount `json:"unitsDelta" doc:"the units added, or taken away when negative"`
	// CostBasis is absent on a sale, whose basis is the average cost worked
	// out on read (ADR 0009).
	CostBasis            *CostBasis `json:"costBasis,omitempty" doc:"the change to the cost of the units held, if the movement states it"`
	OccurredAt           time.Time  `json:"occurredAt"`
	JournalEntryID       *int64     `json:"journalEntryId,omitempty" doc:"the entry of a trade or a vest, which moved the cash"`
	ReversalOfMovementID *int64     `json:"reversalOfMovementId,omitempty"`
	Description          string     `json:"description,omitempty"`
	ExternalID           string     `json:"externalId,omitempty"`
}

// MovementPage is one page of movements, oldest first.
type MovementPage struct {
	Movements []Movement `json:"movements"`
	// Next is the after of the next page, and is absent on the last one.
	Next string `json:"next,omitempty"`
}

// ListMovementsInput is the request of GET /api/holdings/movements.
type ListMovementsInput struct {
	FinancialAccountID int64     `query:"financialAccountId" minimum:"1" doc:"only the movements of this financial account"`
	AssetID            int64     `query:"assetId" minimum:"1" doc:"only the movements of this asset"`
	Kinds              []string  `query:"kind,explode" enum:"trade,vest,split,transfer,adjustment" doc:"only movements of any of these kinds; every kind if omitted"`
	From               time.Time `query:"from" doc:"only movements that occurred at or after this instant"`
	To                 time.Time `query:"to" doc:"only movements that occurred before this instant"`
	Limit              int       `query:"limit" minimum:"1" maximum:"500" doc:"the size of the page; 100 if omitted"`
	After              string    `query:"after" doc:"the next of the previous page; the first page if omitted"`
}

// ListMovementsOutput is the response of GET /api/holdings/movements.
type ListMovementsOutput struct {
	Body MovementPage
}

// newCostBasis is amount as the API returns it.
func newCostBasis(amount lt.CostBasisAmount) CostBasis {
	return CostBasis{CurrencyID: amount.CurrencyID, Amount: gateway.NewAmount(amount.Amount)}
}
