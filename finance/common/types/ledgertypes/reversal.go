package ledgertypes

type ReverseJournalEntryInput struct {
	JournalEntryID int64  `json:"journalEntryId"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type ReverseJournalEntryResult struct {
	JournalEntryID int64 `json:"journalEntryId"`
	// Warnings lists the holdings that are negative after the reversal, which
	// was written regardless.
	Warnings []NegativeHolding `json:"warnings,omitempty"`
}
