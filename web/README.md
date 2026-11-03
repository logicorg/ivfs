# Web UI and HTTP gateway

The web UI covers what is awkward to do through the MCP server: setting up
institutions, accounts, categories, and currencies, reviewing and correcting
entries, and browsing balances. [ADR 0007](../docs/adrs/0007-web-ui-and-http-gateway.md)
records why it is built this way.

It has two halves, built into one binary:

- **The gateway** (`web/main.go`, `web/api`), a Go HTTP API built with
  [Huma](https://huma.rocks) on chi. Each endpoint runs one finance workflow on
  Temporal and returns its result as JSON. It has no database access and no
  rules of its own: validation and invariants stay in the workflows and
  activities.
- **The UI** (`web/ui`), a React app built with Vite, embedded in the gateway
  binary and served at every path that is not under `/api`.

The UI never declares an API type by hand. The gateway generates an OpenAPI
document from its Go types, and the UI generates its TypeScript client from
that document:

```text
Go types in web/api ──make api──▶ web/api/openapi.json ──make api──▶ web/ui/src/api
```

Both generated outputs are committed, and `make api-check` fails when they are
stale.

## Running it

The gateway needs a Temporal server, a finance worker, and an OpenID Connect
provider to sign users in, which the compose setup runs
(`cd compose && docker compose up -d`). The provider is Authelia, on
`https://127.0.0.1:9091`, with three users whose passwords are their names:
`alice` and `bob`, and `eve`, who is not in the `ivfs` group and so is
refused.

| Command | What it does |
| --- | --- |
| `make ui` | Builds the UI into `web/ui/dist/app`, which the gateway embeds the next time it is built. |
| `make web` | Runs the gateway on `http://localhost:8080`, against Temporal on `TEMPORAL_HOST_PORT` or `localhost:7233`, with the compose's Authelia and the manual and the compose's Yahoo asset value providers. |
| `make ui-dev` | Serves the UI with hot reload on `http://localhost:5173`, forwarding `/api` and `/auth` to the gateway. |
| `make api` | Regenerates `openapi.json` and the TypeScript client after an API type changes. |
| `make api-check` | Regenerates both and fails if the committed files differ. |

For UI work, run `make web IVFS_PUBLIC_URL=http://localhost:5173` and
`make ui-dev` side by side, and open `http://localhost:5173`: the gateway
answers only the origin it is told it is served from. A gateway built before
`make ui` answers every UI path with a 503 asking for the build, but its API
works.

The first visit asks the browser to trust Authelia's self-signed certificate.
Signing out of ivfs leaves Authelia's own session, which Authelia 4.39 offers
no way to end from ivfs, so the next visit signs the same user straight back
in. To switch users, also sign out at `https://127.0.0.1:9091/logout`.

### Signing in

Users sign in through an OpenID Connect provider, and each has their own data
([ADR 0013](../docs/adrs/0013-users-and-per-user-databases.md)). The gateway stores no credentials:
it keeps the provider's subject in the registry and the user's id in a signed
session cookie. Every path but `/api/health` and `/auth/*` needs that cookie.
Besides it, the gateway rejects a request whose `Host` it does not answer to,
a request sent from another site (`Origin`, `Sec-Fetch-Site`), and a request
body that is not JSON, so a page open in the same browser cannot drive it.

It is configured from the environment, for which `make web` has development
defaults:

| Variable | What it is |
| --- | --- |
| `IVFS_PUBLIC_URL` | The origin the browser reaches the gateway at, `https://`, or `http://` on localhost. |
| `IVFS_SESSION_KEYS` | Comma-separated base64 keys, each from `openssl rand -base64 32`, that sign session cookies: the first signs, any verifies, so a new key goes first and an old one stays until its sessions expire. |
| `OIDC_ISSUER` | The provider's issuer URL. |
| `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET` | The gateway's client at the provider, which must require PKCE and allow the redirect URI `IVFS_PUBLIC_URL/auth/callback`. |
| `OIDC_GROUP` | Optional: the group a user must be in, from the ID token's `groups` claim. |
| `OIDC_LOGOUT_URL` | Optional: the provider's own sign-out page, for a provider with no `end_session_endpoint`, such as Authelia (`https://auth.example/logout`). Without either, signing out ends the gateway's session only, and the provider signs the user straight back in. |
| `OIDC_CA_FILE` | Optional: PEM certificate authorities to trust for the provider besides the system's, such as Authelia's self-signed one. |
| `IVFS_ASSET_VALUE_PROVIDERS` | Optional: the asset value providers (ADR 0014), in order of preference, as `manual,yahoo=http://yahoo-provider:8080`. `manual` is built in and takes no URL; the others are base URLs of services implementing `docs/specs/asset-value-provider.v1.yaml`. Unset, it is `manual` alone. |
| `IVFS_ASSET_VALUE_PROVIDER_TIMEOUT` | Optional: how long each provider has to answer, as `5s`, the default. |

Authelia's keys and secrets are generated into `compose/authelia/secrets` on
its first start, and its database, `compose/authelia/data`, holds the subject
each user signs in with: delete it and every user comes back as a new, empty
ivfs user. Its certificate lasts a year; delete `tls.crt` and `tls.key` for a
new one.

The UI's tooling is pinned: pnpm 10.34.6 through corepack (the Makefile runs
it from `web/ui`, where `package.json` names that version), and every
dependency at an exact version (`.npmrc` sets `save-exact`). Discuss any
dependency upgrade before making it.

## Layout

```text
web/
  main.go                 the gateway command: serves, or writes the OpenAPI document with -openapi
  api/
    api.go                NewHandler and Spec: wires the areas, the guard, and the UI together
    openapi.json          generated, committed
    gateway/              shared by every area
      workflow.go           RunWorkflow, RunWorkflowWithoutResult, and the error-to-status mapping
      guard.go              the Host, Origin, and JSON checks
      problem.go            RFC 9457 problem responses outside Huma
      amount.go             Amount: an exact decimal, a string on the wire
      integers.go           drops int64's format, so the client uses numbers, not bigints
    apitest/              the API's test helpers
    health/               one package per area, as in mcp/tools:
    currencies/             register.go   Register, which adds the area's endpoints
    categories/             types.go      the area's request and response types
    accounts/               list.go, create.go, ...   one endpoint per file, each with its _test.go
  ui/
    ui.go                 embeds dist and serves it, falling back to index.html
    dist/                 the build (dist/app, ignored) and a committed placeholder
    src/
      api/                generated by hey-api: types, SDK functions, zod schemas, TanStack Query options
      routes/             one file per screen, TanStack Router file routes
      routeTree.gen.ts    generated by the router plugin, committed
      components/         screens' dialogs and shared form fields
      components/ui/      shadcn components, added with the shadcn CLI
      lib/                logic without React, each file with its .test.ts
```

## The gateway

### An endpoint

An endpoint is a file in its area's package that registers one Huma
operation and runs one workflow with `gateway.RunWorkflow`, or
`gateway.RunWorkflowWithoutResult`. When an [ADR 0006](../docs/adrs/0006-intent-level-account-setup.md)
facade does what the screen means to do, the endpoint runs the facade.

```go
func registerCreate(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "create-category",
		Method:      http.MethodPost,
		Path:        "/api/categories",
		Summary:     "Create a category, or find the one with its name under the same parent",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *CreateCategoryInput) (*CreateCategoryOutput, error) {
		result, err := gateway.RunWorkflow[ct.CategoryResult](ctx, c, categorieswf.CreateCategory, ct.CreateCategoryInput{...})
		...
	})
}
```

The conventions:

- **Paths and methods.** Paths start with `/api/`. Reads are `GET` with query
  parameters; commands are `POST` with a JSON body, on a sub-path naming the
  action when it is not a plain create (`/api/categories/{id}/rename`).
- **The operation ID** becomes the TypeScript function's name, so it reads as
  a verb phrase: `list-categories` becomes `listCategories` and
  `listCategoriesOptions`.
- **`Errors`** lists the statuses the endpoint can return besides 200, so the
  document and the client know about them.
- **Errors map to statuses in one place**, `gateway.WorkflowError`: a
  validation error is 422, `ErrNotFound` 404, `ErrResourceConflict` and
  `ErrResourceClosed` 409, and anything else a logged 500. An endpoint does not
  map errors itself.
- **No rules in the endpoint.** It copies its input into the workflow's params
  and the workflow's result into its output, nothing more. A rule that only an
  endpoint enforces is a rule the MCP server and the workflow tests do not see.

### API types

Each area's `types.go` declares its request and response types:

- **Inputs** are local types with Huma's tags (`path`, `query`, `json`,
  `minLength`, ...). The finance types never get HTTP tags; the endpoint
  copies the input into them.
- **Outputs** are flat projections of the workflow's result, with only what a
  screen shows, in camelCase, with enums as string enums.
- **Amounts** are `gateway.Amount`, a string on the wire, so no digit is lost
  to a JavaScript number.
- **Names are unique across areas.** Huma names a schema after its Go type,
  without the package, so two areas cannot both have a `ListOutput` body.
- **Arrays are never null.** The gateway tells Huma they are not nullable, so
  an endpoint returns an empty slice, never nil.

Change a type, then run `make api` and commit the regenerated files with it.

### A new area

Add a package under `web/api` with a `Register(api huma.API, c client.Client)`
that registers its endpoints, and call it from `newAPI` in `api.go`.

## The UI

### A screen

A screen is a file in `src/routes`, which the router plugin turns into a
route (`currencies.tsx` is `/currencies`); add it to the sidebar in
`components/app-sidebar.tsx`. A screen:

- **reads** with the generated query options, such as
  `useQuery(listCategoriesOptions({ query: { includeClosed } }))`;
- **writes** with the generated mutations, such as
  `useMutation(createCategoryMutation())`, and on success invalidates the
  queries the write changed, by their generated keys;
- **keeps its filters in the URL**, with the route's `validateSearch`, so a
  reload or a link keeps them;
- **shows** a pending, an error, and an empty state for every query, the error
  with `ProblemMessages`.

### Forms

Forms use react-hook-form, validated with the generated zod schema of the
request body (`zodResolver(zNewCategory)`), with `TextField` and
`SelectField` from `components/form-fields.tsx`. The schema checks only what
the document says, such as a required field or a length; the workflow still
decides. When the API rejects a request, `lib/problem.ts` puts each of the
problem's errors on its field, and the rest at the top of the form.

### Amounts

An amount stays a string from the API to the screen. `lib/money.ts`
formats it with `Intl.NumberFormat` straight from the string, to the
currency's precision. Never convert an amount to a `number`.

### Components

`components/ui` holds shadcn components, copied into the repository by the
shadcn CLI and not edited by hand. Add one with the CLI, a dependency at the
pinned version: `corepack pnpm@10.34.6 --dir web/ui exec shadcn add <component>`.
It installs the component's own dependencies with `pnpm`, so it needs a `pnpm`
on the PATH.
Screen-specific components, such as a screen's dialogs, go in `components`.

## Testing

| Command | What it runs |
| --- | --- |
| `go test ./web/...` | The gateway's tests: each endpoint against a mocked Temporal client. |
| `golangci-lint run ./web/...` | The Go linters. |
| `make ui-test` | The UI's Vitest tests. |
| `corepack pnpm@10.34.6 --dir web/ui typecheck` | The UI's type check, which `make ui` also runs. |
| `make api-check` | Whether the committed OpenAPI document and client are current. |

How to write the tests:

- [HTTP API tests](../docs/guidelines/tests/http_api.md);
- [UI tests](../docs/guidelines/tests/ui.md);
- and the [common testing practices](../docs/guidelines/tests/common_practices.md)
  both build on.
