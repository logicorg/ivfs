package investmenttypes

import (
	"fmt"
	"slices"

	errs "github.com/logicorg/ivfs/internal/pkg/errors"
)

// Kind is the kind of an investment transaction, which picks the builder that
// turns its facts into an entry and a movement (ADR 0009).
type Kind string

const (
	KindBuy  Kind = "buy"
	KindSell Kind = "sell"
	KindVest Kind = "vest"
)

// kinds lists every kind, in the order an error names them.
var kinds = []Kind{KindBuy, KindSell, KindVest}

// RecordInvestmentInput is one investment transaction: its kind, and the facts
// of that kind in the field named after it. A field per kind, rather than one
// untyped payload, keeps each kind's facts typed and validated by their own
// CUE constraints.
type RecordInvestmentInput struct {
	Kind Kind            `json:"kind" jsonschema:"the kind of the transaction, which names the field holding its facts"`
	Buy  *BuyAssetInput  `json:"buy,omitempty" jsonschema:"the facts of a buy, set when the kind is buy"`
	Sell *SellAssetInput `json:"sell,omitempty" jsonschema:"the facts of a sale, set when the kind is sell"`
	Vest *VestInput      `json:"vest,omitempty" jsonschema:"the facts of a vest, set when the kind is vest"`
}

// Validate checks that the kind is known, that its facts and no other kind's
// are set, and then the facts themselves.
func (i *RecordInvestmentInput) Validate() error {
	if !slices.Contains(kinds, i.Kind) {
		return errs.ErrInvalidParameter{Description: fmt.Sprintf("kind %q is not one of %v", i.Kind, kinds)}
	}

	facts := map[Kind]interface{ Validate() error }{}
	if i.Buy != nil {
		facts[KindBuy] = i.Buy
	}
	if i.Sell != nil {
		facts[KindSell] = i.Sell
	}
	if i.Vest != nil {
		facts[KindVest] = i.Vest
	}

	own, ok := facts[i.Kind]
	if !ok || len(facts) > 1 {
		return errs.ErrInvalidParameter{
			Description: fmt.Sprintf("an investment of kind %q needs its facts in the field %q, and no other kind's", i.Kind, i.Kind),
		}
	}
	return own.Validate()
}
