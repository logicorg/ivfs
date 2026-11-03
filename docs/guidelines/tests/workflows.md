# Workflow testing guidelines

Read the [common testing practices](common_practices.md) first: table-driven
tests by default, no dependency on the order tests run in, error assertions,
and pointers apply here too.

- Use a suite embedding `suite.Suite` and `testsuite.WorkflowTestSuite`.
- Create a fresh `TestWorkflowEnvironment` for every test case.
- Mock activities with `env.OnActivity`, using the concrete activity method and expected arguments.
- A facade (`finance/workflows/facade`) calls workflows, not activities: mock its child workflows with `env.OnWorkflow` instead. Arguments reach a mock decoded from JSON, so a decimal such as `500.50` arrives as `500.5`, equal in value but not field by field; match such an argument with `mock.MatchedBy`.
- Create fixtures for activities when necessary to improve readability and maintainability.
- Prefer table-driven cases covering success, activity failure, and boundary inputs.
    - When not using table-driven tests you must have a strong motivation not to.
- Call `env.ExecuteWorkflow`, then assert `GetWorkflowError()` before decoding a result.
- Decode successful results with `GetWorkflowResult` and compare the complete DTO.
- Do not use a real database in workflow tests; database behavior belongs in activity tests using `TemporalDBUnitTestSuite`. Flows that run workflows against the real activities end to end belong in `finance/setup` or `finance/scenarios`; register everything with `registration.Register`, so a facade finds the workflows it runs as children.
- Remember that Temporal wraps activity errors: casting the workflow error back to the original Go error is not supported. Assert a failure with `require.ErrorContains` on the error's message.

## Single Responsibility Principle

Test ***one and only one*** workflow per test function. Do not test multiple workflows in a single test function (avoid `TestWorkflow1AndWorkflow2`).

Also if the package has multiple workflows in different files, create a separate test file for each workflow file.
Example: if you have `create_account.go` and `delete_account.go` in the same package, create `create_account_test.go` and `delete_account_test.go` for the tests. Do NOT create a single `account_test.go` file for both workflows.
