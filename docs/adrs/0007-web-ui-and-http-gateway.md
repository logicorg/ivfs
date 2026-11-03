# ADR 0007: A web UI over an HTTP gateway to the workflows

- Status: Implemented
- Date: 2026-10-03
- Depends on: [ADR 0002](0002-ledger-domain-and-application-api.md)
- Related: [ADR 0006](0006-intent-level-account-setup.md), whose facades this ADR decides how to expose
- Superseded in part by: [ADR 0013](0013-users-and-per-user-databases.md), for "Security in the first version"

## Context

The MCP server is the only client today. Every operation the system offers
becomes another MCP tool, and a model choosing among many tools does worse than
one choosing among a few. Some operations also suit a conversation poorly:
setting up accounts is done once and is hard to undo, approving imported
movements means checking many rows side by side, and browsing balances and
statements is easier to read as tables and charts than as chat.

So the MCP server keeps what it does well, and a small web UI takes the rest.
The UI needs an HTTP API, since Temporal is an internal orchestration layer and
not something a browser can call: its frontend speaks gRPC, and exposing it
would hand any page full control of every workflow.

## Decision

### Responsibilities

| Client | Owns |
| --- | --- |
| MCP | Reads and questions ("how much did I spend on groceries this quarter?"), importing statements, and quick capture of a single movement. |
| Web UI | Setup: accounts and their balances (`OpenAccount` and `AddBalance`, ADR 0006), categories, and currencies. Review and corrections: approving imported entries, reversing entries, closing accounts. Browsing: statements, balances, holdings, and charts. |

ADR 0006 leaves which client exposes its facades to the client-facing surface;
this is that decision. Setup belongs to the UI: `OpenAccount` and `AddBalance`
are its account form, and ADR 0006's `institution_not_found` question becomes a
dialog. The MCP server does not gain setup tools, and its low-level setup
tools (`create_institution`, `create_financial_account`,
`create_ledger_account`, `create_category`, and `create_currency`) are removed
once the UI covers what they do.

### The workflows stay the backend

A new command, `web/`, is an HTTP gateway and a Temporal client, like `mcp/`.
Each endpoint decodes its request, runs one workflow, and encodes the result.
It adds no business logic and no persistence: validation stays in the
workflows' CUE constraints, and invariants stay in the activities. Where a
facade (ADR 0006) does what a screen means to do, such as opening an account
or creating a category, the endpoint runs the facade rather than the
lower-level workflows it composes, so the order of the steps is never
repeated in the gateway or the UI.

The code that starts a workflow and waits for its result moves out of
`mcp/mcptypes` into a shared package, `internal/pkg/workflowrun`, so both
transports start workflows on the same task queue, with the same start
options, and every endpoint bounds its wait with a `WorkflowExecutionTimeout`
as the MCP tools do.

### The contract: Go types, OpenAPI, generated TypeScript

The Go types are the only hand-written definition of the API:

1. **Huma** (v2), on a chi router, generates an OpenAPI 3.1 document from each
   endpoint's Go input and output types.
2. The document is written to `web/api/openapi.json` by the gateway binary,
   without starting a server.
3. **hey-api** generates the TypeScript types, a typed client, TanStack Query
   hooks, and zod schemas from it, into `web/ui/src/api/`.
4. `make api` runs both steps, and a check runs it and fails if the committed
   files changed, so an API change cannot reach the UI without regenerating
   its types.

Both generated outputs are committed, so a change to the API is visible in
review as a change to `openapi.json`.

Reads are `GET` with query parameters; commands are `POST` with a JSON body.
Paths group endpoints by area, the way `mcp/tools` does: `/api/accounts`,
`/api/currencies`, `/api/journal-entries/{id}/reverse`, and so on.

### API types

- An endpoint uses a workflow's param and result types directly when they are
  flat, such as `ListAccountsParams` and `AccountSummary`.
- It never exposes a `pkg/model` type. Those carry bun relations, which are
  cyclic and describe the database rather than the API; an endpoint whose
  workflow returns one has a flat projection type in the gateway instead.
- A decimal amount is a string in the schema and on the wire, as
  `decimal.Decimal` already marshals it. The gateway's own amount type gives it
  that schema, since `decimal.Decimal` cannot declare one itself. The UI never
  turns an amount into a JavaScript number: it displays the string, and does
  arithmetic with decimal.js.
- A string enum, such as `LedgerAccountKind`, declares its values in the
  schema, so TypeScript sees a union of them rather than `string`.

### Errors

Errors are RFC 9457 problem details, which Huma produces. One mapping turns
the type of a workflow's error into a status. It looks for each type anywhere
in the error's cause chain with `workflows.HasErrorType`, since a facade's
error carries a child workflow's wrapper before the activity's structured
error (ADR 0006):

| Error type | Status |
| --- | --- |
| `ValidationError`, `ErrInvalidParameter`, `ErrMissingRequiredParameter` | 422 |
| `ErrNotFound` | 404 |
| `ErrResourceConflict`, `ErrResourceClosed` | 409 |
| anything else, including a timed-out workflow | 500, logged with the workflow ID |

An outcome that asks the caller a question, such as `OpenAccount`'s
`institution_not_found`, is a 200 response, as it is a result and not an error
(ADR 0006). The UI answers it with a dialog and sends the request again.

### The UI

| Concern | Choice |
| --- | --- |
| Language and build | TypeScript, React, Vite |
| Components | shadcn/ui, on Tailwind CSS |
| Server state | TanStack Query, through the generated hooks |
| Tables | TanStack Table |
| Forms | react-hook-form, validated with the generated zod schemas |
| Charts | Recharts, which shadcn/ui's chart components wrap |
| Routing | TanStack Router, whose typed search parameters hold a screen's filters |
| Package manager | pnpm |

The zod schemas only give early feedback in the form. The workflows' CUE
constraints remain the validation that counts, and their 422 is shown on the
form.

### Layout and delivery

```
web/
  main.go         flags, the Temporal client, the router, the embedded UI
  api/            Huma endpoints by area, each with its tests, and openapi.json
  ui/             the Vite project; src/api/ is generated
```

The production build of `web/ui` is embedded in the gateway binary with
`go:embed` and served at `/`, with the API under `/api`, so the UI is one
binary and one compose service. During development, Vite's dev server serves
the UI and proxies `/api` to the gateway.

### Security in the first version

The UI has a single user and no authentication. The gateway listens on
`127.0.0.1` only, and guards against another site's page in the same browser
calling it:

- it accepts request bodies only as `application/json`, which a cross-site
  form cannot send without a CORS preflight, and it answers no preflight;
- it rejects a request whose `Sec-Fetch-Site` or `Origin` names another site;
- it rejects a request whose `Host` is not the address it listens on, which
  stops DNS rebinding.

Authentication is decided when the UI is reached from another device.

### Tests

- Gateway endpoints are tested table-driven, with `httptest` and the mocked
  Temporal client from `go.temporal.io/sdk/mocks`, as
  `internal/pkg/workflowrun/workflowrun_test.go` does: each case asserts the
  workflow's params, the status, and the body. The error mapping has a case per row of
  the table above.
- The drift check above covers the contract.
- UI logic that can be wrong on its own, such as amount formatting and the
  handling of a two-step outcome, is kept out of components and tested with
  Vitest. Component tests and end-to-end tests are deferred; the
  [UI testing guide](../guidelines/tests/ui.md) records the choices made for
  when they come.
- The [HTTP API testing guide](../guidelines/tests/http_api.md) details the
  gateway's tests.

## Alternatives considered

- **Go only: templ with shadcn-templ and htmx.** One language, no API contract
  to keep in sync, and amounts that never leave Go. Rejected because the
  screens that matter most, reviewing imports and browsing statements, need
  rich tables, which React's libraries already provide; because shadcn-templ
  2.0 is a beta with one maintainer, and every screen would depend on it; and
  because it is the stack Claude, which builds much of this UI, knows least.
- **`html/template` and htmx.** Too little structure: forms, layouts, and
  components would all be written by hand.
- **Leptos.** Exact decimals and Rust's type system, but a thin component
  ecosystem, a pre-1.0 API, and a second compiled language.
- **A CLI or terminal UI instead of a web UI.** The cheapest option, needing no
  gateway at all, and well suited to setup. Rejected for charts and for
  reviewing many rows, and because it cannot be reached from another device.
- **The browser calling Temporal's HTTP API directly.** It would expose full
  control of every workflow to any page, take several calls per operation, and
  return domain errors as raw Temporal failures.
- **A Node backend (Next.js) with Temporal's TypeScript client.** It would
  start workflows by name across a language boundary, and repeat the types,
  the error mapping, and the timeouts in TypeScript.
- **Spec-first generators: oapi-codegen, ogen, goa.** Healthy projects, but the
  Go types they generate would need mapping to the existing ones, so the types
  would exist twice.
- **Other code-first generators.** Fuego has not released since February 2025.
  swaggest/openapi-go fits technically, but has one maintainer and would sit
  in the request path. A registry of our own on `google/jsonschema-go`, which
  the MCP SDK already uses, would share one schema source with MCP, but we
  would maintain the document assembly, and it is unproven that it names its
  types well enough for TypeScript.
- **tygo, with no OpenAPI.** Go structs become TypeScript interfaces directly,
  but paths, parameters, and errors stay hand-written, and there are no
  generated hooks or zod schemas.

## Consequences

- The repository gains a second language and a Node toolchain, used only to
  build `web/ui`.
- A change to an API type takes a `make api` run, and the generated files show
  up in its diff.
- Huma becomes a dependency of the gateway's request path. Its minor releases
  have tightened behaviour before (stricter validation of `email` and `uri`
  formats, required form data by default), so its version is pinned exactly
  and its changelog read before an upgrade.
- Descriptions in the API schema use Huma's `doc` tag, a third tag dialect next
  to `cue` and `jsonschema`. They are optional, since the UI does not need
  them.
- The gateway embeds `web/ui`'s build output, so `go:embed` needs it to exist.
  A placeholder in the build directory keeps `go build ./...` and
  `go test ./...` working on a fresh checkout; a release builds the UI first.
- Until the UI's setup screens exist, setting up accounts still goes through
  the MCP server's low-level tools.
- Reviewing imported entries waits for the draft journal lifecycle in
  `docs/todos.md`, which needs its own decision. Charts may need reporting
  queries that do not exist yet.
- The design relies on these minimum versions: Huma v2, OpenAPI 3.1 (for the
  generators), and Tailwind CSS v4 (for current shadcn/ui). Exact versions are
  pinned in `go.mod` and `pnpm-lock.yaml`, not here.
