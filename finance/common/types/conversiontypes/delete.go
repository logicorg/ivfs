package conversiontypes

import "time"

// DeleteRequest defines the parameters for deleting a currency conversion.
type DeleteRequest struct {
	ID           int64
	FromCurrency int64
	ToCurrency   int64
	ObservedAt   time.Time
}

// DeleteResponse defines the response for deleting a currency conversion.
type DeleteResponse struct{}
