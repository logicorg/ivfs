package gateway

import (
	"context"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/client"
)

// AccountDeletionPreview is what deleting an account or an institution would
// do, read when asked (ADR 0012).
type AccountDeletionPreview struct {
	DeletedEntries        int `json:"deletedEntries" doc:"journal entries deleted with the account"`
	KeptEntries           int `json:"keptEntries" doc:"journal entries kept, with the account's lines moved to the untracked accounts"`
	DeletedMovementGroups int `json:"deletedMovementGroups" doc:"position movement groups deleted with the account"`
	KeptMovementGroups    int `json:"keptMovementGroups" doc:"position movement groups kept, with the account's legs moved to the untracked accounts"`
	// StrayAmounts is what lands on each untracked or uncategorized ledger
	// account.
	StrayAmounts []StrayAmount `json:"strayAmounts,omitempty"`
	// StrayUnits is what lands on the untracked financial account, by asset.
	StrayUnits []StrayUnits `json:"strayUnits,omitempty"`
	Conflicts  []string     `json:"conflicts,omitempty" doc:"why the deletion would be refused"`
}

// StrayAmount is what lands on one untracked or uncategorized ledger account.
type StrayAmount struct {
	Name         string                  `json:"name"`
	Kind         model.LedgerAccountKind `json:"kind"`
	CurrencyCode string                  `json:"currencyCode"`
	Amount       Amount                  `json:"amount" doc:"debits minus credits"`
}

// StrayUnits is how many units of an asset land on the untracked financial
// account.
type StrayUnits struct {
	AssetID   int64  `json:"assetId"`
	AssetName string `json:"assetName"`
	Units     Amount `json:"units"`
}

// AccountDeletionPreviewOutput is the response of a deletion preview.
type AccountDeletionPreviewOutput struct {
	Body AccountDeletionPreview
}

// PreviewDeletion runs the preview workflow wf with in and returns its
// preview as the API answers it.
func PreviewDeletion(ctx context.Context, c client.Client, wf, in any) (*AccountDeletionPreviewOutput, error) {
	preview, err := RunWorkflow[lt.AccountDeletionPreview](ctx, c, wf, in)
	if err != nil {
		return nil, err
	}
	return &AccountDeletionPreviewOutput{Body: NewAccountDeletionPreview(preview)}, nil
}

// NewAccountDeletionPreview is preview as the API returns it.
func NewAccountDeletionPreview(preview *lt.AccountDeletionPreview) AccountDeletionPreview {
	out := AccountDeletionPreview{
		DeletedEntries: preview.DeletedEntries, KeptEntries: preview.KeptEntries,
		DeletedMovementGroups: preview.DeletedMovementGroups, KeptMovementGroups: preview.KeptMovementGroups,
		Conflicts: preview.Conflicts,
	}
	for _, amount := range preview.StrayAmounts {
		out.StrayAmounts = append(out.StrayAmounts, StrayAmount{
			Name: amount.Name, Kind: amount.Kind, CurrencyCode: amount.CurrencyCode, Amount: NewAmount(amount.Amount),
		})
	}
	for _, units := range preview.StrayUnits {
		out.StrayUnits = append(out.StrayUnits, StrayUnits{AssetID: units.AssetID, AssetName: units.AssetName, Units: NewAmount(units.Units)})
	}
	return out
}
