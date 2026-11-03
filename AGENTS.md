# IVFS
Go + Temporal + PostgreSQL personal-finance backend built on a double-entry ledger.
Goal: manage cashflows and investments across currencies with production-grade
accuracy (exact decimals, fractional shares). Rationale lives in `docs/`. Read the relevant parts before changing code.

## Communication
- Dry, concise, low-key humor. No flattery, no preambles or postambles.
- Comments explain "why", not "what".
- Error messages: actionable and specific.
- Work in small batches.

## Activities and workflows
- Activities own their SQL transaction (`db.RunInTx`) end to end and are idempotent.
- Creation is retrieve-or-insert: on retry return the existing entity unchanged, never reconcile it. A change is an explicit `Update` from the workflow.

## Errors
- Domain failures use the generic, resource-parameterized types in `internal/pkg/errors`,
  never `errors.New`/`fmt.Errorf` or per-entity types.
- Wrap only infrastructure errors (`fmt.Errorf("insert x: %w", err)`).

## Package layout
- One operation per file (`create.go`, `retrieve.go`, ...), each with a matching
  `_test.go`. Don't merge operations, or tests of different workflows.
- Split crowded packages by domain, e.g. `finance/activities/ledger/journal` and
  `finance/activities/ledger/portfolio`.

## Testing
- Table-driven. Cases are order-independent: each creates its fixtures, runs, deletes
  them (`docs/guidelines/tests/common_practices.md`).
- Activities: `temporaltesting.TemporalDBUnitTestSuite` on a real PostgreSQL container,
  at most one per package (`docs/guidelines/tests/activities.md`). Cover edge cases.
  Never use one activity to set up or tear down another's test.
- Fixtures: generic helpers in `internal/pkg/temporaltesting/model_helpers.go`
  (`Insert[M]`, `InsertOrRetrieve[M]`, `Retrieve[M]`, `Delete[M]`, `DeleteIgnoreNotFound[M]`).
  Don't extend `InsertAsset`/`InsertCurrency`; entity-specific setup goes in
  `internal/pkg/temporaltesting`.
- Mutating activities need an idempotency test: run twice with the same input, assert
  result and persisted state. Read-only ones (`Get*`, `List*`, balances) are exempt.
- Workflows: fixtures for input, activities mocked with `env.OnActivity`, no database
  (`docs/guidelines/tests/workflows.md`). One workflow per test function and per test file.
