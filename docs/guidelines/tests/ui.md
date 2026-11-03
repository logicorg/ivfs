# Web UI testing guidelines

Read the [common testing practices](common_practices.md) first; they apply
here too.

The UI is a thin client (ADR 0007): the rules live in the workflows and are
tested in Go. UI tests cover only what the UI can get wrong on its own, such as
formatting what the API returns or mapping its errors to form fields.

## Today: pure logic only

Tests are Vitest tests of plain functions in `src/lib`, run in Node, with no
DOM. Keep components thin and move any branch worth testing into `src/lib`,
since components are not tested yet. `groupByInstitution` and the accounts
route's `validateSearch` are candidates.

- A test sits next to its file: `money.ts` and `money.test.ts`.
- Compare the whole result, with `toEqual` or `toBe`.
- Pass anything machine-dependent explicitly, such as the locale.
- Amounts are decimal strings, never numbers, in fixtures too.

Tests are table-driven, with a typed array of named cases and a loop (not
`it.each`):

```ts
describe("formatAmount", () => {
  const cases: { name: string; amount: string; code: string; want: string }[] = [
    { name: "a negative amount", amount: "-0.01", code: "USD", want: "-$0.01" },
    { name: "a code Intl rejects", amount: "1200.5", code: "USDT", want: "1,200.5 USDT" },
  ]

  for (const { name, amount, code, want } of cases) {
    it(name, () => {
      expect(formatAmount(amount, code, undefined, "en-US")).toBe(want)
    })
  }
})
```

Build each case's fixtures in the table or inside `it`, never in a
module-level variable a test changes.

## What not to test

- Generated code: `src/api` and `routeTree.gen.ts`.
- shadcn components in `src/components/ui`, and Radix behind them.
- Rules the API enforces.
- Libraries' own behaviour, markup and styling. No snapshot tests.

## Commands

```sh
make ui-test                                                        # run every test
corepack pnpm@10.34.6 --dir web/ui exec vitest                      # watch mode
corepack pnpm@10.34.6 --dir web/ui exec vitest run --sequence.shuffle  # expose order dependence
corepack pnpm@10.34.6 --dir web/ui typecheck
```

## Later

Decided on 2026-10-03:

- **Component and screen tests: not yet.** When they come, they:
  - fake the API with MSW 3.0.2 and handwritten handlers, typed with the
    generated response types;
  - find elements by role and name, and interact with `userEvent`;
  - use a fresh `QueryClient` per test.

  The DOM is still to choose. Vitest browser mode is recommended over jsdom,
  because Radix needs browser APIs that jsdom lacks.
- **End-to-end tests: after the open-account wizard.** They would be a few
  Playwright flows against the real stack, run with a separate `make ui-e2e`.
- **No coverage target.**
