package transactions

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	ledgerfxwf "github.com/logicorg/ivfs/finance/workflows/ledger/fx"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// transferKind prefixes a transfer's idempotency key, in either currency case.
const transferKind = "transfer"

// RecordTransfer records money moved from one balance to another: the balance
// it leaves is credited and the one it reaches debited (ADR 0008). Paying a
// card off is a transfer to its liability, so the amount owed shrinks.
//
// Between balances in different currencies the money that arrives is given as
// ToAmount, and the transfer is posted through RegisterFXConversion, each
// currency balancing through its "FX clearing" account. It fails with
// ErrInvalidParameter, writing nothing, when an account is not a balance, or
// when the amounts do not fit the balances' currencies. A repeated idempotency
// key returns the entry first posted with it.
func RecordTransfer(ctx workflow.Context, input tt.RecordTransferInput) (*tt.RecordResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	accounts, err := retrieveLedgerAccounts(ctx, input.FromLedgerAccountID, input.ToLedgerAccountID)
	if err != nil {
		return nil, err
	}
	from, to := accounts[0], accounts[1]
	if err := checkBalance(from); err != nil {
		return nil, err
	}
	if err := checkBalance(to); err != nil {
		return nil, err
	}

	if from.CurrencyID == to.CurrencyID {
		if !input.ToAmount.IsZero() && !input.ToAmount.Equal(input.Amount) {
			return nil, internalerrors.ErrInvalidParameter{
				Description: fmt.Sprintf("%q and %q are in the same currency, so the amount that arrives is the amount that leaves", from.Name, to.Name),
			}
		}
		return post(ctx, movement{
			kind:           transferKind,
			idempotencyKey: input.IdempotencyKey,
			from:           from.ID,
			to:             to.ID,
			amount:         input.Amount,
			occurredAt:     input.OccurredAt,
			description:    input.Description,
		})
	}

	if input.ToAmount.IsZero() {
		return nil, internalerrors.ErrInvalidParameter{
			Description: fmt.Sprintf("%q and %q are in different currencies; give the amount that arrives as toAmount", from.Name, to.Name),
		}
	}
	return convert(ctx, input, from, to)
}

// convert posts a transfer between currencies as an FX conversion, with no
// fee. The clearing accounts are found by their key, or created if a balance
// was opened without them, as account setup would have.
func convert(ctx workflow.Context, input tt.RecordTransferInput, from, to *model.LedgerAccount) (*tt.RecordResult, error) {
	fromClearing, err := ensureFXClearing(ctx, from.CurrencyID)
	if err != nil {
		return nil, err
	}
	toClearing, err := ensureFXClearing(ctx, to.CurrencyID)
	if err != nil {
		return nil, err
	}

	conversion := lt.RegisterFXConversionInput{
		IdempotencyKey:                     transferKind + ":" + input.IdempotencyKey,
		OccurredAt:                         input.OccurredAt,
		SourceLedgerAccountID:              from.ID,
		SourceClearingLedgerAccountID:      fromClearing,
		SourceAmount:                       input.Amount,
		DestinationLedgerAccountID:         to.ID,
		DestinationClearingLedgerAccountID: toClearing,
		DestinationAmount:                  input.ToAmount,
	}
	if input.Description != "" {
		conversion.Description = &input.Description
	}

	var result lt.RegisterFXConversionResult
	if err := workflow.ExecuteChildWorkflow(ctx, ledgerfxwf.RegisterFXConversion, conversion).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("post %s: %w", transferKind, err)
	}
	return &tt.RecordResult{JournalEntryID: result.JournalEntryID}, nil
}

// ensureFXClearing returns the id of the "FX clearing" account of a currency.
func ensureFXClearing(ctx workflow.Context, currencyID int64) (int64, error) {
	var created lt.CreateLedgerAccountResult
	err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.CreateLedgerAccount, lt.CreateLedgerAccountInput{
		CurrencyID: currencyID, Name: lt.FXClearingAccountName, Kind: model.LedgerAccountAsset,
	}).Get(ctx, &created)
	if err != nil {
		return 0, fmt.Errorf("find the FX clearing account of currency %d: %w", currencyID, err)
	}
	return created.LedgerAccountID, nil
}
