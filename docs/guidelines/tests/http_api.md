# HTTP API testing guidelines

Read the [common testing practices](common_practices.md) first; they apply
here too.

The web API (`web/api`, ADR 0007) only translates: each endpoint decodes a
request, runs one workflow, and encodes the result or the error. Its tests
check that translation, with `httptest` and a mocked Temporal client: no
database, no Temporal server.

## What to test, and what not

An endpoint test checks that:

- the request becomes exactly the params the workflow is given;
- the workflow's result becomes the body the UI expects;
- a request the input types forbid is rejected before any workflow runs;
- a workflow's error reaches the UI with its status and message.

It does not test rules (a name taken, a move not allowed, CUE constraints,
idempotency), which belong to workflow and activity tests, nor what every
endpoint shares (the error table, the guard, timeouts, `Amount`), which is
tested once in `gateway`.

## Shape

One test file per endpoint file, one `TestX` per endpoint, with a doc comment
saying what it checks. Cases are a table run by `apitest.RunCases`:

```go
// TestUpdate checks that only the fields in the body reach UpdateCurrency,
// and that an invalid id runs no workflow.
func TestUpdate(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"rename a currency": {
			Method: http.MethodPatch, Path: "/api/currencies/3", Body: `{"name":"Euro"}`,
			Call: &apitest.WorkflowCall{
				Workflow: currencieswf.UpdateCurrency,
				Params:   &ct.UpdateCurrencyParams{ID: 3, Name: ptr.NewPtr("Euro")},
			},
			WantStatus: http.StatusNoContent,
		},
		"an invalid id": {
			Method: http.MethodPatch, Path: "/api/currencies/0", Body: `{"name":"Euro"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
```

A `WorkflowCall` expects exactly one run of that workflow with those params;
any other workflow fails the test, and a case with no `Call` asserts that none
runs.

- **Params and results are complete values**, and `Params` is of the
  workflow's parameter type (a pointer if the workflow takes one).
- **Send what the UI sends**: literal paths and a JSON string body.
- **Assert every body** with `WantBody`, using `apitest.Problem(status,
  detail)` for a problem. A `204` has none.
- **Build errors as the workflow returns them**:
  `temporal.NewApplicationError(msg, "ErrNotFound")`, or
  `apitest.ChildWorkflowError(type, msg)` for a facade's child workflow.

An endpoint that runs no workflow, such as health, uses `apitest.Serve`
in its own table.

## Minimum cases per endpoint

- A success for each mapping: every optional query parameter or field, once
  present and once absent.
- An empty list, answered as `[]`.
- The wire form of amounts (exact digits, a negative one), enums, and times.
- One case per status in the operation's `Errors`.
- One case, with no `Call`, per constraint the input types declare.

## What `apitest` checks for you

- **The data converter.** The endpoint's params are encoded with Temporal's
  default data converter and decoded into the workflow's parameter type, and
  the result goes through it the other way, as with a real client and worker.
  So `Params` is written in the workflow's parameter type, and a field that
  does not survive JSON fails the test.
- **The schema.** Every successful body is validated against its operation's
  response schema in the OpenAPI document (`huma.Validate`), which catches a
  `null` array or a malformed `Amount` even in a case without `WantBody`.

## The contract

`TestSpecIsUpToDate` fails when `openapi.json` is stale, and `make api-check`
when the TypeScript client is. Never edit either by hand; run `make api` and
commit the result with the change.

## Running

```sh
go test -count=3 ./web/...
golangci-lint run ./web/...
```

## Not adopted

- Asserting Huma's full 422 body, and the problem content type on every error.
- `-race` in `make test`.
- humatest: it sets no `Host`, which the guard needs, and marshals bodies the
  cases deliberately write as JSON.
- An integration test against a real Temporal: deferred, with the UI's
  end-to-end tests, until the open-account wizard.
