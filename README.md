# IVFS

IVFS is a Temporal-based personal-finance backend built around a double-entry
ledger. It is currently pre-alpha; the database can be recreated from scratch
and no migration compatibility is provided.

## Model

The source of truth for monetary events is `ledger.journal_entry` plus its
`ledger.posting` rows. Investment units are tracked separately in
`ledger.position_movement`. A category is a row of `ledger.category`, with a
bounded hierarchy and no currency: it classifies a posting to the "Expenses"
or "Income" ledger account of a currency, and holds no money itself.
Tags are flat, optional annotations in `tag.tag`.

The database schemas are:

- `account` — financial accounts;
- `asset` — securities and other assets;
- `currency` — currencies and observed conversions;
- `institution` — banks, brokers, and other providers;
- `ledger` — financial accounts, ledger accounts, categories, counterparties,
  journal entries, postings, and position movements; and
- `tag` — flat tags.

See [ADR 0001](docs/adrs/0001-double-entry-ledger.md), [ADR 0002](docs/adrs/0002-ledger-domain-and-application-api.md), [ADR 0003](docs/adrs/0003-categories-and-tags.md), and [ADR 0019](docs/adrs/0019-currency-neutral-categories.md) for the design decisions.

## Development

Run the worker with the repository's compose setup:

```sh
cd compose
docker compose up -d
```

Each user gets a database of their own on first sign-in ([ADR 0013](docs/adrs/0013-users-and-per-user-databases.md)).
A `compose/psqldata` from before users has no `ivfs_app` role, so the worker
cannot start; create it with
`docker compose exec postgres-ivfs /docker-entrypoint-initdb.d/ivfs-init.sh`.
The single-user data in its `postgres` database is no longer read by anything.

The web UI and its HTTP gateway are described in [web/README.md](web/README.md).

Run the Go test suite with:

```sh
go test ./...
```

The integration tests use the Temporal/PostgreSQL test scaffolding described
in [the activity testing guide](docs/guidelines/tests/activities.md), and the
practices every test follows in
[the common testing practices](docs/guidelines/tests/common_practices.md).
