# ADR 0013: Users, each with their own database

- Status: Implemented
- Date: 2026-10-09
- Supersedes: the "Security in the first version" part of [ADR 0007](0007-web-ui-and-http-gateway.md)
- Related: [ADR 0005](0005-uniqueness-and-idempotent-creation.md), whose keys stay
  as they are; [ADR 0012](0012-deleting-accounts.md), whose deletion workflow IDs
  become per user

## Context

ivfs has one user. The gateway listens on loopback with no authentication
(ADR 0007), and the worker, the MCP server, and the gateway share one database.
Reaching the UI from another device, or letting a second person use it, needs
users. The requirements are:

- each user is completely separate from the others: no shared accounts, no
  shared data, reference data (currencies, assets, tags) included;
- ivfs stores no user credentials;
- a user can never start a workflow that acts on another user's data. A
  ledger that one user's request corrupts in another's database cannot be
  repaired by looking at it, so this rules out designs that are correct only
  as long as every query remembers a filter.

Two parts of the current design matter. Deletions run under workflow IDs
derived from what they delete (`DeleteLedgerAccountWorkflowID(42)`) and join a
run already under that ID (ADR 0012). If ids repeat across users, user A's
deletion of account 42 would join user B's run. And every unique key and
idempotency key (ADR 0005) is global: in a shared table, one user's "Revolut"
would block another's.

## Decision

### Identity: OIDC in the gateway

The gateway signs users in with OpenID Connect against an external identity
provider (Authelia), using the authorization code flow with PKCE (S256),
`state`, and `nonce`. A user is identified by the ID token's `(issuer,
subject)`. ivfs never sees a password, and it stores none of the IdP's tokens:
the ID token is verified at the callback and discarded.

Who may sign up is the IdP's decision: anyone it authenticates gets an account.
Configuration can also require a group claim.

### Storage: one database per user

Each user has their own PostgreSQL database, `ivfs_u_<uid>`, where `uid` is
the user's id as 32 hex characters. Inside it the schema, the triggers, the
ADR 0005 keys, and the deletion settings (ADR 0011, ADR 0012) are exactly what
they are today. Separation does not depend on any query remembering a filter,
because a query can only see the database its connection belongs to.

A registry database, `ivfs`, holds one table:

| Column | Meaning |
| --- | --- |
| `id` | UUIDv7, the user's id; never reused |
| `issuer`, `subject` | the OIDC identity, unique together |
| `created_at` | when the row was inserted |
| `provisioned_at` | when the user's database was ready; null until then |

The database name is derived from `id`, not stored. Ids are UUIDs rather than
a sequence so that they are never handed out twice, even if the registry is
recreated while user databases, workflow IDs, or session cookies from before
it still exist. Version 7 keeps them in creation order.

User databases are owned by a role `ivfs_app`, which has `CREATEDB` but is not
a superuser. The `admin` superuser only creates that role and the registry.

### Provisioning: `EnsureUser`

At each login, after verifying the ID token, the gateway runs the `EnsureUser`
workflow with the token's `(issuer, subject)`. Its activities are idempotent:

1. retrieve-or-insert the registry row, which yields the uid;
2. create the user's database unless `pg_database` already lists it.
   `CREATE DATABASE` cannot run in a transaction, so a concurrent login can
   race it. `duplicate_database` (42P04) counts as success, and so does
   `unique_violation` (23505), which is what the loser of a race past
   PostgreSQL's own name check gets;
3. run `sqllib.InitDatabase` in it, then set `provisioned_at`.

`EnsureUser` runs on its own task queue, `provisioning`, served by a worker
that can reach only the registry. It is the only workflow that runs without a
user.

### Carrying the user to the activities

A package, `tenancy`, owns the user in a Go context. Its context key is
unexported, so only its own functions can set or read it. A Temporal
`ContextPropagator` copies the user into a workflow header when the gateway
starts a workflow, and back into the workflow context and each activity's
context on the worker. Child workflows, activities, and continue-as-new carry
it the same way.

Activities hold a `Pools` instead of a `*bun.DB`, and each starts with
`a.Pools.For(ctx)`, which returns the connection pool of the user in `ctx`.
`For` fails when `ctx` has no user and when the user's `provisioned_at` is
null. It runs `InitDatabase` the first time a process opens a user's pool, so
a schema change reaches every user's database. The user never travels as a
workflow or activity argument, so no input type can be filled in with someone
else's id.

### Workflow IDs are per user

Every workflow the gateway starts runs under `<uid>/<id>`:
`workflowrun.Start` takes the uid from `ctx`, fails without one, and prefixes
the ID it was given, or a random one. `ExecuteDeletionChild` prefixes its
child's ID in the same way, using the user in the workflow context. User A's
deletion of account 42 joins `A/delete-ledger-account-42` and never B's run.
The `Delete*WorkflowID` functions are unchanged. The 202 response to a
deletion strips the prefix, so the API does not change.

### The trust boundary

The user in a header is not signed. The worker would have to hold the key to
sign the workflows it starts itself, which would let it mint any claim; a
signed claim cannot expire while a long deletion is still retrying; and it
would sit in every workflow's history. Instead:

- **The network.** The gateway is the only Temporal client. Temporal's
  frontend port (7233) and its UI are reachable only on the internal network.
  The development compose publishes them, and both databases, on loopback
  only: the gateway runs on the host there, so the host is that network.
- **The code.** A forbidigo rule allows `tenancy.WithUser` only in the
  gateway's authentication middleware and in tests.
- **The worker fails closed, three times.**
  1. An inbound interceptor on the finance worker rejects, as non-retryable,
     any workflow or activity without a user.
  2. It also rejects a workflow whose root workflow ID (its own when it is the
     root) does not start with `<uid>/` for the user in its header, which
     catches a start that bypassed `workflowrun`.
  3. `Pools.For` refuses a context without a user, or one whose database is
     not provisioned.

### Sessions

After the callback the gateway sets its own session cookie, `__Host-session`,
holding `{uid, exp}` and signed with HMAC-SHA256. It is `Secure`, `HttpOnly`,
and `SameSite=Lax`. A session lasts 12 hours and is never reissued. Signing
keys come from configuration as a list: the first signs, any of them verifies,
so a key can be rotated without signing everyone out. The uid in the cookie is
not a secret; the signature is what makes it trustworthy.

The cookie is stateless, so a session cannot be revoked before it expires,
except by rotating every key out. Logout clears the cookie and sends the
browser to the IdP's `end_session_endpoint`, when the IdP has one.

While the browser is away at the IdP, `state`, `nonce`, and the PKCE verifier
wait in a second signed cookie, which lives 10 minutes and is sent only to
`/auth/callback`.

Every path except `/auth/*` and `/api/health` needs a session; the UI sends
the browser to `/auth/login` when the API answers 401. The ADR 0007 guard
stays, as the defence against cross-site requests, and it now accepts the
configured public host rather than loopback names.

### MCP is deferred

The MCP server reaches Temporal directly over stdio and has no user. Once the
worker enforces users, its calls are rejected. Serving MCP over HTTP from the
gateway, with OAuth against the same IdP, is a later decision.

### Existing data

There is one user's data today, and it is test data. It is dropped, not
migrated: the first login provisions an empty database. Deleting a user is out
of scope, and will be `DROP DATABASE` plus a mark in the registry.

## Alternatives considered

- **One shared database with row-level security.** Every table gains a
  `user_id`, a policy, and composite foreign keys. It is a single database to
  run, but its safety rests on configuration that must never slip: `FORCE ROW
  LEVEL SECURITY` on every table, a role that neither owns the tables nor
  bypasses RLS, `user_id` leading every unique key and idempotency key, and no
  `SECURITY DEFINER` function. Each would need a test of its own. It also
  changes every ADR 0005 key, and deleting a user would fight the triggers
  that keep the ledger immutable.
- **A task queue per user.** Each user's worker is bound to their database at
  registration, so activities would not change, and a misrouted task finds no
  worker. Rejected because it needs a worker per user, created when the user
  is provisioned, and still needs per-user workflow IDs.
- **A Temporal namespace per user.** As above, and workflow IDs and histories
  are separated for free. Rejected for its cost: a namespace registered for
  each user, and a client per namespace in the gateway.
- **The uid as a workflow argument.** Visible and simple, but every input type
  would carry it, and nothing would stop a caller from passing someone else's.
- **Signed user headers.** See "The trust boundary".
- **Forward authentication at the reverse proxy**, with the gateway trusting a
  `Remote-User` header. Less code, but the gateway's safety would then depend
  on a proxy's configuration it cannot check.
- **A sequence for user ids.** Shorter names, but a recreated registry hands
  out the same ids again.

## Consequences

- Every activity changes from `a.DB` to `a.Pools.For(ctx)`: 44 files, a
  mechanical change. Activity tests build `tenancy.Static(s.DB)`, which
  returns the test database for any context, and routing gets tests of its
  own.
- Each user's database costs a pool on the worker. That is fine for a few
  users; many would need an idle pool to close, or PgBouncer.
- A schema change reaches a user's database the next time a worker process
  opens it, not at deploy time.
- The worker gains an interceptor and a propagator, which must stay registered
  on every worker and client. A test pins that the user survives
  continue-as-new.
- Running ivfs locally needs an IdP: the development compose file gains
  Authelia.
- The MCP server stops working until it moves behind the gateway.
- The design relies on PostgreSQL 15 or later (ADR 0005), Temporal Go SDK
  v1.49 (`RootWorkflowExecution` in `WorkflowInfo`), go-oidc v3, and
  `golang.org/x/oauth2` with PKCE support. Exact versions live in `go.mod`.
