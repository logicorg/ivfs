# Design Principles

This document outlines the key design principles and architectural patterns observed in the `ivfs` codebase. Adhering to these principles will help maintain the consistency, stability, and maintainability of the project.

## Ledger Invariants

- `JournalEntry` and immutable `Posting` rows are the monetary source of
  truth. Do not reintroduce a two-sided `Transaction` table.
- Every posted journal entry balances debits and credits independently for
  each currency.
- `LedgerAccount` is the balance-bearing chart of accounts. It has one
  currency and one accounting kind: asset, liability, equity, income, or
  expense.
- A category is not a table or kind. It is an income or expense ledger
  account, exposed through the small `categories` workflow facade.
- Facades under `finance/workflows/facade` present what a person means to do,
  such as opening an account, and compose the lower-level workflows as child
  workflows rather than calling their activities.
- `FinancialAccount` represents an institution relationship; it is not a
  balance. Its linked ledger accounts represent concrete balances such as
  Revolut EUR or IBKR USD cash.
- `PositionMovement` is the source of truth for investment units. Monetary
  trades create both postings and a position movement in one SQL transaction.
- FX uses currency-specific clearing accounts. Fees and rounding differences
  are explicit postings, never hidden in a rate or silently discarded.
- Activities own database transactions and financial write invariants.
  Workflows only orchestrate one atomic activity per financial write.

The detailed rationale is in [ADR 0001](adrs/0001-double-entry-ledger.md),
[ADR 0002](adrs/0002-ledger-domain-and-application-api.md),
[ADR 0003](adrs/0003-categories-and-tags.md),
[ADR 0005](adrs/0005-uniqueness-and-idempotent-creation.md), and
[ADR 0006](adrs/0006-intent-level-account-setup.md).
Ideas still under discussion are in [`docs/rfcs`](rfcs/README.md).

## Idempotency in Creation Activities

Creation activities must be idempotent to prevent duplicate records when an activity is retried due to transient failures.

The standard approach in this project is **"retrieve-or-insert"**.

1.  The activity first attempts to retrieve the entity based on a unique identifier (e.g., a name or a composite key).
2.  If the entity is found, the activity immediately returns the existing entity's data without performing any modifications. It does **not** compare or update other properties.
3.  If the entity is not found, the activity proceeds with creating the new entity.

This pattern ensures that retrying a creation activity results in a single, consistent record. It prioritizes simplicity and predictability over reconciling potential differences in the request parameters of a retried execution. If an update is needed, a dedicated `Update` activity should be explicitly called by the workflow.

## Immutability of Core Financial Attributes

Certain core attributes of financial models are treated as immutable after creation. This is a deliberate design choice to ensure data integrity and simplify the system's logic.

A key example is the `Currency` of an `Account`. The `UpdateAccount` function does not allow changing an account's currency. This is because changing the currency of an account with existing transactions would have complex implications for historical data and reporting. Instead of performing a complex and risky data migration, the system requires the user to create a new account with the desired currency.

## Separation of Concerns

The codebase is structured to separate different aspects of the application, making it easier to understand, maintain, and test. The main components of this architecture are:

*   **`activities`**: Contain the core business logic for specific operations (e.g., creating an account, inserting a transaction).
*   **`workflows`**: Orchestrate multiple activities to perform higher-level business processes.
*   **`model`**: Defines the core data structures and their relationships.
*   **`common/types`**: Contains the objects used for input and output of activities and workflows.
*   **`pkg/sqllib`**: A library for database interactions, abstracting away the underlying SQL driver.

## Structured Error Handling

The project uses a custom error package located in `internal/pkg/errors`. This allows for the use of structured, specific errors instead of generic ones. For example, the system uses `errors.ErrMissingRequiredParameter` and `errors.ErrInvalidParameter` to provide clear feedback about what went wrong. This makes debugging and error handling more robust.

## Abstracted Database Interaction

Database operations are abstracted using a query builder or ORM-like library (`pkg/sqllib`). Instead of writing raw SQL queries, the code uses methods like `a.DB.NewInsert()` and `a.DB.NewSelect()`. This has several advantages:
*   It reduces the risk of SQL injection vulnerabilities.
*   It makes the code more readable and easier to write.
*   It allows for easier migration to a different database system in the future.

## Input Sanitization

All inputs to activities are validated at the beginning of the function. For example, the `validateInsertAccountParameters` function in the `InsertAccount` activity validates required parameters, checks for invalid values, and sets default values. This ensures that the business logic only has to deal with valid and consistent data.

Use CUE-lang constraints and cuego where possible to validate inputs.

## Testing

The project aims for a comprehensive test suite. The presence of `_test.go` files for many of the activities indicates a commitment to unit testing. The testing strategy includes:
* **Unit tests**: Each activity has its own set of tests to verify its functionality in isolation.
* **Test helpers**: The `internal/pkg/temporaltesting` package provides helper functions for testing Temporal workflows and activities, and `pkg/model/testhelpers` provides helpers for creating model objects in tests. This reduces code duplication in tests and makes them easier to write.
* **Table-driven tests**: use table-driven testing as much as possible. It makes it easier to setup and grasp what each test is trying to achieve.
* **Fixtures**: use fixtures for workflows testing to setup the activities that the workflow calls whenever it makes sense.

# Activities And Workflow Pacakages Structuring

Each type of operation should be in its own separate file (e.g. `insert.go`, `retrieve.go`, `delete.go`, etc.).

This allows to keep each file content cohese, and allows for easier readability and maintainability. It also makes the repo easier to navigate.

Each of this file should have its own related test file (e.g. `insert_test.go`, `retrieve_test.go`, `delete_test.go`, etc.).
