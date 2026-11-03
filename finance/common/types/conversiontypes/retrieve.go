package conversiontypes

import (
	"time"

	"cuelang.org/go/cuego"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// RetrieveRequest defines the parameters for retrieving currency conversions.
type RetrieveRequest struct {
	FromCurrency  int64     `cue:">0"`
	ToCurrency    int64     `cue:">0"`
	Date          time.Time `cue:"!=\"\""`
	WithRelations bool
}

func (r *RetrieveRequest) Validate() error {
	err := cuego.Validate(r)
	if err != nil {
		return errors.ValidationError{Err: err}
	}
	return nil
}

// RetrieveResponse defines the response for retrieving currency conversions.
type RetrieveResponse struct {
	Conversions  []*model.Conversion
	FromCurrency *model.Currency
	ToCurrency   *model.Currency
}
