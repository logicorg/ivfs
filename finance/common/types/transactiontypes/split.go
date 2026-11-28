package transactiontypes

import (
	"fmt"

	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// Split is part of an amount booked on "Expenses" under a category of its own:
// a line of an expense paid for several things, or a charge of a transfer, such
// as an instalment's interest, a fee, or an early-repayment penalty.
type Split struct {
	CategoryID int64           `json:"categoryId" jsonschema:"the expense category it is booked under"`
	Amount     decimal.Decimal `json:"amount" jsonschema:"the part of the amount it takes, positive"`
}

// SplitsTotal is what the splits take of an amount.
func SplitsTotal(splits []Split) decimal.Decimal {
	total := decimal.Zero
	for _, split := range splits {
		total = total.Add(split.Amount)
	}
	return total
}

// validateSplits checks that each split, a line or a charge as noun names it,
// has a category and a positive amount.
func validateSplits(noun string, splits []Split) error {
	for _, split := range splits {
		if split.CategoryID <= 0 {
			return errs.ErrInvalidParameter{Description: fmt.Sprintf("a %s names the expense category it is booked under", noun)}
		}
		if !split.Amount.IsPositive() {
			return errs.ErrInvalidParameter{Description: fmt.Sprintf("a %s is a positive part of the amount, not %s", noun, split.Amount)}
		}
	}
	return nil
}

// validateCharges checks the charges of a transfer, and that something is left
// to reach the other balance; a payment that is all interest is an expense, not
// a transfer.
func validateCharges(amount decimal.Decimal, charges []Split) error {
	if err := validateSplits("charge", charges); err != nil {
		return err
	}
	if total := SplitsTotal(charges); len(charges) > 0 && !total.LessThan(amount) {
		return errs.ErrInvalidParameter{
			Description: fmt.Sprintf("the charges add up to %s of the %s that leaves, so nothing reaches the other balance; record it as an expense", total, amount),
		}
	}
	return nil
}

// validateLines checks that an expense names either its category or the lines
// it was spent on, and that the lines add up to its amount.
func validateLines(categoryID int64, amount decimal.Decimal, lines []Split) error {
	if len(lines) == 0 {
		if categoryID <= 0 {
			return errs.ErrInvalidParameter{Description: "an expense names the categoryId it was spent on, or its lines"}
		}
		return nil
	}
	if categoryID != 0 {
		return errs.ErrInvalidParameter{Description: "an expense with lines takes its categories from them; omit categoryId"}
	}
	if err := validateSplits("line", lines); err != nil {
		return err
	}
	if total := SplitsTotal(lines); !total.Equal(amount) {
		return errs.ErrInvalidParameter{Description: fmt.Sprintf("the lines add up to %s, not the %s paid", total, amount)}
	}
	return nil
}
