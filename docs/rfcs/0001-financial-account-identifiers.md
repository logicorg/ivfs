# RFC 0001: Recording account identifiers

- Status: Under discussion
- Date: 2026-10-03
- Related: [ADR 0005](../adrs/0005-uniqueness-and-idempotent-creation.md) (`FinancialAccount.ExternalID`), [ADR 0006](../adrs/0006-intent-level-account-setup.md) (opening accounts)

## Problem

Statement imports need to know which account a statement belongs to, and
statements identify the account by its details: an IBAN, a sort code and
account number, a US routing and account number, a broker account number, or
the last digits of a card. So the details have to be recorded somewhere they
can be matched.

The model has one place for them today, `FinancialAccount.ExternalID`: one
optional value per financial account, compared exactly and unique within its
institution (ADR 0005). Real accounts do not all fit that:

| Account | Its details |
| --- | --- |
| Revolut, personal | Separate details per currency: an IBAN for EUR, a sort code and account number for GBP, a routing and account number for USD. A statement per currency. |
| A Swiss multi-currency bank account | One IBAN for every currency. |
| Interactive Brokers | One account number for every currency and every holding. |
| A Revolut savings vault | No details of its own: it lives inside the EUR account. |
| A credit card | A card number, of which only the last four digits should be stored. Two cards at the same issuer can share them. |

The question is where an IBAN goes when a financial account holds several
currencies. Putting it on the "Cash" ledger account was considered and rejected
straight away: a ledger account is an accounting balance, not an address. A
vault has no IBAN, one IBAN can cover several balances, and a broker account
number covers all of them.

## Option A: identifiers as rows of their own

A financial account has any number of identifiers, each optionally limited to
one currency:

```text
account.financial_account_identifier
  financial_account_id   the account it identifies, such as Revolut Personal
  scheme                 iban | sort_code_account | aba_routing_account | broker_account | card_last4
  value                  "LT12 3456 ..."
  currency_id            EUR, or NULL when it covers every currency of the account
```

Revolut Personal has three rows, one per currency; the Swiss account one IBAN
row with no currency; IBKR one `broker_account` row; a card one `card_last4`
row, which is a hint for matching rather than a unique key.

An import matches the statement's identifier to its row, takes the financial
account from it and the currency from the row or else from the statement, and
posts to that account's main balance in that currency.

- **For:** one financial account per real relationship with an institution,
  as the onboarding docs model it; several identifiers per account (an IBAN
  and a local sort code for the same GBP balance); schemes are explicit, so a
  card hint is never mistaken for a unique key.
- **Against:** a new table, a new uniqueness rule (on scheme and value, or on
  value alone?), and an extra step between a statement and its balance.
  `FinancialAccount.ExternalID` becomes redundant and would be dropped.

## Option B: one financial account per set of details

A financial account is defined as one set of account details, which is what
one statement covers, and `FinancialAccount.ExternalID` stays the only place
an identifier is recorded:

- Revolut is opened as one financial account per currency: "Revolut Personal
  EUR", "Revolut Personal USD", each with its own IBAN or account number.
- The Swiss account is one financial account with several currencies, under
  its single IBAN.
- IBKR is one financial account with its account number.
- A vault is a balance inside "Revolut Personal EUR".

An import matches the statement's identifier to the financial account and the
statement's currency to its balance.

- **For:** no new table or rule; `ExternalID` and its unique index already
  exist; matching is direct; the rule is the same for every institution rather
  than a Revolut special case.
- **Against:** nothing groups "Revolut Personal EUR" with "Revolut Personal USD"
  apart from the institution and their names, so a personal and a business
  account at the same institution are told apart only by name; one identifier
  per account, so when an account has two (an IBAN and a sort code), only the
  one its statements use is recorded; moving money between Revolut currencies
  is a transfer between financial accounts, which conversions already support.

## Where things stand

Neither option is chosen. [ADR 0006](../adrs/0006-intent-level-account-setup.md)
opens accounts without capturing identifiers, so `ExternalID` stays empty for
now, and the choice is left to the design of statement imports.

Still open:

- Option A or Option B, and whether Option B's single `ExternalID` is enough
  in practice, for example for an account with two identifiers.
- Whether matching uses the value alone or the scheme and the value.
- Whether a card's last four digits are recorded at all, and how an import
  tells apart two cards that share them.
- How identifiers are captured: an optional field when opening an account, a
  separate operation to add one, or both.
- Whether identifiers are normalised before they are compared, such as an IBAN
  with and without spaces.
