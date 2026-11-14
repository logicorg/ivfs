# Testing: common practices

These practices apply to every test in the repository. The layer-specific
guides build on them:

- [Activity tests](activities.md): a real PostgreSQL database, one activity at a
  time.
- [Workflow tests](workflows.md): activities mocked, no database.
- [HTTP API tests](http_api.md): the web gateway's endpoints, Temporal client
  mocked.
- [UI tests](ui.md): the web UI, in TypeScript; the table-driven and
  independence rules apply there too.

## Table-driven tests by default

Write a table-driven test by default, even for two or three cases: "found" and
"not found", or "missing name" and "name taken", are one table, not two test
functions. Give every case a descriptive name, and run it as a subtest.

Keep case fields plain values. A value that is known when the table is built
is written as a value, not wrapped in a function: reference data inserted into
a fresh schema, such as the first two currencies, always gets IDs 1 and 2, so
insert it at the top of the test function and use `&eur.ID` in the table, or
name the IDs as constants. Every function field is indirection a reader has to
resolve.

A case field may be a small function only when the input depends on a fixture
that each case creates, whose ID changes from case to case:

```go
tests := map[string]struct {
	// findBy selects the institution to update, given the fixture.
	findBy func(institution *model.Institution) it.RetrieveInstitutionParams
	name   string
}{
	"by id": {
		findBy: func(i *model.Institution) it.RetrieveInstitutionParams { return it.RetrieveInstitutionParams{ID: i.ID} },
		name:   "investments",
	},
	"by name": {
		findBy: func(i *model.Institution) it.RetrieveInstitutionParams { return it.RetrieveInstitutionParams{Name: i.Name} },
		name:   "savings",
	},
}
```

Write a separate test function instead only when the cases genuinely need
different setup or different assertions that a table field cannot express
cleanly. Idempotency tests, which execute the same operation twice and then
inspect the state, are always their own function, and so is a single
end-to-end flow.

## No test depends on the order tests run in

A test or a table case that passes only because something else ran before it,
or fails only because something else did, is a bug. Order-dependent
tests pass or fail at random, and may hide real regressions. Go iterates a map
in random order, so a map-based table exposes an order dependency sooner or later.

To keep cases independent:

- **Give each case its own fixtures**, created for that case and deleted after
  it, not created once before the loop and shared. Reference data that every
  test in a suite needs, such as currencies, belongs in the suite's
  `BeforeTest`, which runs before every test.
- **Never capture a value from a per-case fixture into the table**, such as an
  ID when the table is built. Derive it from the fixture the case was given,
  with a function field as above. Fixtures created once per test function,
  before the table, are not per-case and can be used directly.
- **Delete what each case created once the case has run**, even when it
  failed, so the next case starts from the same state.
- **Do not paper over leftover rows** with an upsert such as
  `temporaltesting.InsertOrRetrieve`. It hides the rows a previous case left
  behind instead of preventing them, and it breaks as soon as the uniqueness
  rule it relies on changes.

### The default shape: insert, run, delete

In the loop, create the case's fixture, run the case as a subtest, and delete
the fixture after the subtest returns:

```go
for name, tc := range tests {
	institution := &model.Institution{Name: "cash"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	s.Run(name, func() {
		res, err := s.Env.ExecuteActivity(s.institution.Update, it.UpdateInstitutionParams{FindBy: tc.findBy(institution), Name: tc.name})
		// ...
	})

	temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Institution)(nil), institution.ID)
}
```

The delete runs even when the case fails: a failed `require` stops only the
subtest, and the loop carries on. The suite's `AfterTest` drops every table at
the end of the test function in any case. Tests here never run in parallel,
so the delete always runs after the case has finished.

Use `DeleteIgnoreNotFound`, since a case may itself delete its fixture.

When a case genuinely has to create its fixture inside the subtest, register
the delete as a cleanup there instead. A cleanup must not query with the
finished subtest's context: Go cancels a test's context just before running its
cleanups, so the query fails with `context canceled` and leaves its rows
behind. Pass the parent test, `s.T()`, which is the suite's test again by the
time the cleanup runs:

```go
s.Run(name, func() {
	institution := &model.Institution{Name: "cash"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	s.T().Cleanup(func() {
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Institution)(nil), institution.ID)
	})
	// ...
})
```

### Check it

Run a changed package several times before handing it off, which varies the
order of map-based table cases:

```sh
go test -count=3 ./finance/activities/institution/
```

## Assert errors by their message

Assert a domain error with `require.ErrorContains` on the structured error's
message, such as `"resource Currency not found"` or `"resource Currency
conflict"`. Do not unwrap a `*temporal.ApplicationError` to compare its
`Type()`.

## Truncate a time you compare with a stored one

PostgreSQL stores a timestamp to the microsecond. `time.Now()` reads
nanoseconds on Linux and microseconds on macOS, so a test that compares it with
the row read back passes on a Mac and fails in CI. Use a fixed `time.Date`, or
`time.Now().UTC().Truncate(time.Microsecond)`.

## Initialise pointers with `pkg/ptr`

Use `ptr.NewPtr` to take a pointer to a value, and `ptr.Int64` for an `*int64`.
Never take the address of a local variable inline.

## Fixing a bug: the test first

When a test exposes a bug, commit the failing test first and the fix
separately, so the history shows the bug and what fixed it, and the fix can be
discussed before it is written.
