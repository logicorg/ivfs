# Activity testing guidelines

Activity tests verify the activity's real behavior against its dependencies, especially
database persistence. They are not workflow tests: use a Temporal activity test
environment, a real test database, and call only the activity under test.

Read the [common testing practices](common_practices.md) first: table-driven
tests by default, no dependency on the order tests run in, cleanups, error
assertions, and pointers apply here too.

## Suite setup

Embed `temporaltesting.TemporalDBUnitTestSuite` in a package-level
`UnitTestSuite`. The shared suite owns the PostgreSQL test container and database.
Keep one container per package.

```go
type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite

	aa *AssetActivity
}

func (s *UnitTestSuite) SetupTest() {
	s.SetupTestEnvironment()
	s.aa = &AssetActivity{DB: s.DB}
	s.Env.RegisterActivity(s.aa)
}

func (s *UnitTestSuite) BeforeTest(_, _ string) {
	sqllib.InitDatabaseT(s.T(), s.DB)
}

func (s *UnitTestSuite) AfterTest(_, _ string) {
	s.DropAllTables()
}

func TestUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}
```

`SetupTest` creates a fresh `TestActivityEnvironment` and registers the concrete
activity. `BeforeTest` initializes the schema for the test and `AfterTest` removes
it. Use `s.T().Context()` for database operations so test cancellation is respected.

## Arrange data with `temporaltesting`

Fixtures must be created directly through `temporaltesting.Insert` or the relevant
`temporaltesting` helper (`InsertAsset`, `InsertCurrency`, and so on). Helpers may
also be used to clean up fixtures, for example `Delete` or
`DeleteIgnoreNotFound`.

An activity test **must not call another activity to populate or clean the
database**. Doing so couples tests to another activity's implementation and makes
the tests interdependent. If a fixture helper is missing, add a small database
helper to `internal/pkg/temporaltesting`; do not invoke an activity as setup.

## Execute only the activity under test

Invoke the registered activity through `s.Env.ExecuteActivity`. Assert the returned
error before asserting the result, and verify database state directly with queries.
Do not test workflow orchestration or mock the activity's database operations here.

```go
temporaltesting.Insert(s.T(), s.DB, asset)

result, err := s.Env.ExecuteActivity(
	s.aa.UpdateAsset,
	assettypes.UpdateAssetParams{
		FindBy: assettypes.RetrieveAssetParams{ID: asset.ID},
		Name:   ptr.NewPtr("new name"),
	},
)
require.NoError(t, err)
require.Empty(t, result)

var stored model.Asset
require.NoError(t,
	s.DB.NewSelect().Model(&stored).
		Where("id = ?", asset.ID).
		Scan(s.T().Context()),
)
assert.Equal(t, "new name", stored.Name)
```

## Table-driven cases

Cases are table-driven by default, as the [common practices](common_practices.md)
describe. In an activity test, the loop creates each case's fixture, runs the
case, and deletes the fixture, and the case derives any parameter that depends
on it (such as its ID) from that fixture, so that no case depends on another
having run first. Include both successful and failing inputs.

```go
tests := map[string]struct {
	// params builds the parameters, given the fixture.
	params func(asset *model.Asset) assettypes.DeleteAssetParams
	err    bool
}{
	"by id":       {params: func(a *model.Asset) assettypes.DeleteAssetParams { return assettypes.DeleteAssetParams{ID: a.ID} }},
	"by name":     {params: func(a *model.Asset) assettypes.DeleteAssetParams { return assettypes.DeleteAssetParams{Name: a.Name} }},
	"invalid id":  {params: func(*model.Asset) assettypes.DeleteAssetParams { return assettypes.DeleteAssetParams{ID: -1} }, err: true},
	"no selector": {params: func(*model.Asset) assettypes.DeleteAssetParams { return assettypes.DeleteAssetParams{} }, err: true},
}

for name, tc := range tests {
	asset := &model.Asset{Name: "cash", Kind: model.AssetKindCash}
	temporaltesting.Insert(s.T(), s.DB, asset)

	s.Run(name, func() {
		_, err := s.Env.ExecuteActivity(s.aa.DeleteAsset, tc.params(asset))
		if tc.err {
			require.Error(s.T(), err)
			return
		}
		require.NoError(s.T(), err)
	})

	temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Asset)(nil), asset.ID)
}
```

Include edge cases such as invalid or missing identifiers, empty required fields, non-existent records, boundary values, duplicate input, and conflicting
relationships.

For each case, assert the complete externally visible outcome:
the error message when relevant, returned values, and the resulting database
state. A failure case should also verify that no unintended mutation occurred.

## Idempotency

Activities that can be mutate the database (or the application state in general) must have an explicit idempotency test. Execute the
same activity with the same arguments at least twice and assert that both calls
have the expected result. Query the database afterward to ensure the second call
did not create duplicates, apply a mutation twice, or corrupt state.

```go
s.InsertAsset(s.T(), asset)
params := assettypes.DeleteAssetParams{ID: asset.ID}

_, err := s.Env.ExecuteActivity(s.aa.DeleteAsset, params)
require.NoError(s.T(), err)

_, err = s.Env.ExecuteActivity(s.aa.DeleteAsset, params)
require.NoError(s.T(), err)
```

Do not assume idempotency from a successful second call: verify the persisted
state and any affected rows.

Activities that do not need an idempotency test are those that are read-only, e.g.:
- `GetAsset` – get operations
- `ListAssets` – list operations 
- `GetAssetBalance` – balance operations.

## Checklist

- [ ] The test uses `UnitTestSuite` and `SetupTestEnvironment`.
- [ ] The schema is initialized with `sqllib.InitDatabaseT`.
- [ ] Fixtures use only `temporaltesting` helpers or direct fixture SQL helpers.
- [ ] No other activity is called by setup, teardown, or the test.
- [ ] Cases are table-driven unless there is a clear reason otherwise.
- [ ] Happy paths, invalid input, missing data, and boundary cases are covered.
- [ ] Database state is asserted directly after the activity executes.
- [ ] Retryable behavior has an explicit idempotency test.
- [ ] Each case gets its own fixtures, deleted after it runs, and does not rely on execution order.
- [ ] The package passes when run several times (`go test -count=3`).