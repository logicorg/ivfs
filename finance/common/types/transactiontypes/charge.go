package transactiontypes

import (
	"fmt"

	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// Charge is part of a transfer's amount spent on moving it rather than
// reaching the other balance: an instalment's interest, a fee, an
// early-repayment penalty. It is booked on "Expenses" in the source currency,
// under its category.
type Charge struct {
	CategoryID int64           `json:"categoryId" jsonschema:"the expense category it is booked under"`
	Amount     decimal.Decimal `json:"amount" jsonschema:"the part of the amount it takes, positive"`
}

// ChargesTotal is what the charges take of a transfer's amount.
func ChargesTotal(charges []Charge) decimal.Decimal {
	total := decimal.Zero
	for _, charge := range charges {
		total = total.Add(charge.Amount)
	}
	return total
}

// validateCharges checks that each charge names a category and takes a
// positive part of the amount, and that something is left to reach the other
// balance; a payment that is all interest is an expense, not a transfer.
func validateCharges(amount decimal.Decimal, charges []Charge) error {
	for _, charge := range charges {
		if charge.CategoryID <= 0 {
			return errs.ErrInvalidParameter{Description: "a charge names the expense category it is booked under"}
		}
		if !charge.Amount.IsPositive() {
			return errs.ErrInvalidParameter{Description: fmt.Sprintf("a charge is a positive part of the amount, not %s", charge.Amount)}
		}
	}
	if total := ChargesTotal(charges); len(charges) > 0 && !total.LessThan(amount) {
		return errs.ErrInvalidParameter{
			Description: fmt.Sprintf("the charges add up to %s of the %s that leaves, so nothing reaches the other balance; record it as an expense", total, amount),
		}
	}
	return nil
}
