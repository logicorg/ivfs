package journal

import (
	fxt "github.com/logicorg/ivfs/finance/common/types/fxtypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/gateway"
	"github.com/logicorg/ivfs/web/api/transactions"
)

// EntryTransaction is a journal entry as the transaction forms show it and the
// endpoints that correct an entry take it. Exactly one field is set.
type EntryTransaction struct {
	Expense  *transactions.Expense  `json:"expense,omitempty"`
	Income   *transactions.Income   `json:"income,omitempty"`
	Transfer *transactions.Transfer `json:"transfer,omitempty"`
}

// transactionOf reads entry back as the expense, income, or transfer whose
// postings it has, or returns nil when no transaction posts them: a draft,
// which a correction refuses, a trade or a vest, whose units a form cannot
// show, an opening balance, a refund credited to "Expenses", or an entry built
// by hand. A balance is an asset or liability of a financial account, and an
// asset of none is an "FX clearing" account, only found in an FX conversion.
func transactionOf(entry *model.JournalEntry) *EntryTransaction {
	if entry.Status != model.JournalEntryPosted {
		return nil
	}
	var balances, clearing, expenses, incomes []model.Posting
	for _, posting := range entry.Postings {
		account := posting.LedgerAccount
		switch {
		case account.IsBrokerageInvestments():
			return nil
		case account.Kind == model.LedgerAccountExpense:
			if posting.Side != model.PostingDebit || posting.CategoryID == nil {
				return nil
			}
			expenses = append(expenses, posting)
		case account.Kind == model.LedgerAccountIncome:
			if posting.Side != model.PostingCredit || posting.CategoryID == nil {
				return nil
			}
			incomes = append(incomes, posting)
		case (account.Kind == model.LedgerAccountAsset || account.Kind == model.LedgerAccountLiability) && account.FinancialAccountID != nil:
			balances = append(balances, posting)
		case account.Kind == model.LedgerAccountAsset:
			clearing = append(clearing, posting)
		default:
			return nil
		}
	}
	fx := entry.Metadata["type"] == fxt.FXConversionEntryType
	if fx != (len(clearing) > 0) {
		return nil
	}

	var description string
	if entry.Description != nil {
		description = *entry.Description
	}
	var tagIDs []int64
	for _, tag := range entry.Tags {
		tagIDs = append(tagIDs, tag.ID)
	}

	switch {
	case !fx && len(balances) == 1 && balances[0].Side == model.PostingCredit && len(expenses) > 0 && len(incomes) == 0:
		expense := &transactions.Expense{
			BalanceLedgerAccountID: balances[0].LedgerAccountID, Amount: gateway.NewAmount(balances[0].Amount),
			OccurredAt: entry.OccurredAt, Description: description, TagIDs: tagIDs,
		}
		if len(expenses) == 1 {
			expense.CategoryID = *expenses[0].CategoryID
		} else {
			expense.Lines = splitsOf(expenses)
		}
		return &EntryTransaction{Expense: expense}
	case !fx && len(balances) == 1 && balances[0].Side == model.PostingDebit && len(incomes) == 1 && len(expenses) == 0:
		return &EntryTransaction{Income: &transactions.Income{
			BalanceLedgerAccountID: balances[0].LedgerAccountID, CategoryID: *incomes[0].CategoryID,
			Amount: gateway.NewAmount(balances[0].Amount), OccurredAt: entry.OccurredAt, Description: description, TagIDs: tagIDs,
		}}
	case len(balances) == 2 && len(incomes) == 0 && (!fx || len(expenses) <= 1):
		from, to := balances[0], balances[1]
		if from.Side == model.PostingDebit {
			from, to = to, from
		}
		if from.Side != model.PostingCredit || to.Side != model.PostingDebit {
			return nil
		}
		transfer := &transactions.Transfer{
			FromLedgerAccountID: from.LedgerAccountID, ToLedgerAccountID: to.LedgerAccountID,
			Amount: gateway.NewAmount(from.Amount), OccurredAt: entry.OccurredAt, Description: description, TagIDs: tagIDs,
			Charges: splitsOf(expenses),
		}
		if fx {
			transfer.ToAmount = gateway.NewAmount(to.Amount)
		}
		return &EntryTransaction{Transfer: transfer}
	}
	return nil
}

// splitsOf are the "Expenses" debits as lines or charges, nil for none.
func splitsOf(postings []model.Posting) []transactions.SplitForm {
	if len(postings) == 0 {
		return nil
	}
	splits := make([]transactions.SplitForm, len(postings))
	for i, posting := range postings {
		splits[i] = transactions.SplitForm{CategoryID: *posting.CategoryID, Amount: gateway.NewAmount(posting.Amount)}
	}
	return splits
}
