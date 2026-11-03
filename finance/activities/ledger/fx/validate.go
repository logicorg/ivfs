package fx

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// accountRole is the part an account plays in a conversion and what that
// part requires of the account.
type accountRole struct {
	name       string
	account    *model.LedgerAccount
	kind       model.LedgerAccountKind
	currencyID int64
}

// validateAccounts checks the conversion against the accounts it names: every
// account exists, the two sides use different currencies, and each account
// plays a role that matches its kind and currency. Posting rejects closed
// accounts, so that is not checked here.
func validateAccounts(input lt.RegisterFXConversionInput, accounts map[int64]*model.LedgerAccount) error {
	for _, id := range input.AccountIDs() {
		if _, ok := accounts[id]; !ok {
			return internalerrors.ErrNotFound{ResourceType: "LedgerAccount", ID: id}
		}
	}

	source := accounts[input.SourceLedgerAccountID]
	destination := accounts[input.DestinationLedgerAccountID]
	if source.CurrencyID == destination.CurrencyID {
		return internalerrors.ErrInvalidParameter{Description: "source and destination ledger accounts must have different currencies"}
	}

	roles := []accountRole{
		{"source", source, model.LedgerAccountAsset, source.CurrencyID},
		{"destination", destination, model.LedgerAccountAsset, destination.CurrencyID},
		{"source clearing", accounts[input.SourceClearingLedgerAccountID], model.LedgerAccountAsset, source.CurrencyID},
		{"destination clearing", accounts[input.DestinationClearingLedgerAccountID], model.LedgerAccountAsset, destination.CurrencyID},
	}
	if input.Fee != nil {
		roles = append(roles, accountRole{"fee", accounts[input.Fee.LedgerAccountID], model.LedgerAccountExpense, source.CurrencyID})
	}

	for _, role := range roles {
		if role.account.Kind != role.kind {
			return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("%s ledger account must be of kind %s", role.name, role.kind)}
		}
		if role.account.CurrencyID != role.currencyID {
			return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("%s ledger account must be denominated in the currency of its side of the conversion", role.name)}
		}
	}
	return nil
}
