import { useQueries, useQuery } from "@tanstack/react-query"
import { createFileRoute, Link } from "@tanstack/react-router"
import { ChevronLeft, ChevronRight } from "lucide-react"

import {
  getAccountBalancesOptions,
  getCategoryTotalsOptions,
  listCategoriesOptions,
  listCurrenciesOptions,
  listFinancialAccountsOptions,
} from "@/api/@tanstack/react-query.gen"
import { ProblemMessages } from "@/components/form-fields"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableFooter, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { formatAmount } from "@/lib/money"
import { netWorth } from "@/lib/net-worth"
import { addMonths, formatMonth, isMonth, monthOf, monthRange, periodTotals, type Month } from "@/lib/period-totals"

// OverviewSearch is the month the income and expenses are of, kept in the
// URL; the current one if omitted.
type OverviewSearch = { month?: Month }

export const Route = createFileRoute("/")({
  validateSearch: (search: Record<string, unknown>): OverviewSearch =>
    isMonth(search.month) ? { month: search.month } : {},
  component: Overview,
})

function Overview() {
  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">Overview</h1>
      <NetWorth />
      <IncomeAndExpenses />
    </div>
  )
}

// NetWorth totals the balances of every open account by currency. It asks
// for each account's balances as the accounts screen does, so the two share
// what they have loaded.
function NetWorth() {
  const accounts = useQuery(listFinancialAccountsOptions({ query: {} }))
  const currencies = useQuery(listCurrenciesOptions())
  const balances = useQueries({
    queries: (accounts.data?.financialAccounts ?? []).map((account) =>
      getAccountBalancesOptions({ path: { id: account.id } }),
    ),
  })

  const precisions = new Map(currencies.data?.currencies.map((currency) => [currency.code, currency.precision]))
  const failed = balances.find((query) => query.isError)
  const totals = netWorth(balances.flatMap((query) => query.data?.balances ?? []))

  return (
    <Card>
      <CardHeader>
        <CardTitle>Net worth</CardTitle>
        <CardDescription>
          What your open <Link to="/accounts" className="underline">accounts</Link> hold, less what their cards owe,
          in each currency.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {accounts.isPending || balances.some((query) => query.isPending) ? (
          <p className="text-muted-foreground">Loading…</p>
        ) : accounts.isError ? (
          <ProblemMessages error={accounts.error} />
        ) : failed ? (
          <ProblemMessages error={failed.error} />
        ) : totals.length === 0 ? (
          <p className="text-muted-foreground">No balances yet.</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Currency</TableHead>
                <TableHead className="text-right">Held</TableHead>
                <TableHead className="text-right">Owed</TableHead>
                <TableHead className="text-right">Net</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {totals.map(({ currencyCode, held, owed, net }) => {
                const format = (amount: string) => formatAmount(amount, currencyCode, precisions.get(currencyCode))
                return (
                  <TableRow key={currencyCode}>
                    <TableCell className="text-muted-foreground">{currencyCode}</TableCell>
                    <TableCell className="text-right tabular-nums">{format(held)}</TableCell>
                    <TableCell className="text-right tabular-nums">{format(owed)}</TableCell>
                    <TableCell className="text-right font-medium tabular-nums">{format(net)}</TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}

// IncomeAndExpenses shows what each top-level category earned or spent in a
// month, with its subcategories rolled up into it, by currency.
function IncomeAndExpenses() {
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const thisMonth = monthOf(new Date())
  const month = search.month ?? thisMonth
  const totals = useQuery(getCategoryTotalsOptions({ query: monthRange(month) }))
  const categories = useQuery(listCategoriesOptions({ query: { includeClosed: true } }))
  const currencies = useQuery(listCurrenciesOptions())

  const precisions = new Map(currencies.data?.currencies.map((currency) => [currency.code, currency.precision]))
  const periods = periodTotals(totals.data?.totals ?? [], categories.data?.categories ?? [])
  const show = (next: Month) => navigate({ search: next === thisMonth ? {} : { month: next } })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Income and expenses</CardTitle>
        <CardDescription>{formatMonth(month)}</CardDescription>
        <CardAction>
          <Button variant="ghost" size="icon" aria-label="Previous month" onClick={() => show(addMonths(month, -1))}>
            <ChevronLeft />
          </Button>
          <Button variant="ghost" size="icon" aria-label="Next month" onClick={() => show(addMonths(month, 1))}>
            <ChevronRight />
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        {totals.isPending || categories.isPending ? (
          <p className="text-muted-foreground">Loading…</p>
        ) : totals.isError ? (
          <ProblemMessages error={totals.error} />
        ) : categories.isError ? (
          <ProblemMessages error={categories.error} />
        ) : periods.length === 0 ? (
          <p className="text-muted-foreground">Nothing earned or spent this month.</p>
        ) : (
          periods.map(({ currencyCode, income, expenses, net, rows }) => {
            const format = (amount: string) => formatAmount(amount, currencyCode, precisions.get(currencyCode))
            return (
              <Table key={currencyCode}>
                <TableHeader>
                  <TableRow>
                    <TableHead>Category</TableHead>
                    <TableHead className="w-40 text-right">{currencyCode}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rows.map((row) => (
                    <TableRow key={row.id}>
                      <TableCell>
                        <span className="flex items-center gap-2">
                          {row.name}
                          {row.kind === "income" && <Badge variant="secondary">income</Badge>}
                        </span>
                      </TableCell>
                      <TableCell className="text-right tabular-nums">{format(row.total)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
                <TableFooter>
                  <TableRow>
                    <TableCell>Income</TableCell>
                    <TableCell className="text-right tabular-nums">{format(income)}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell>Expenses</TableCell>
                    <TableCell className="text-right tabular-nums">{format(expenses)}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell>Net</TableCell>
                    <TableCell className="text-right tabular-nums">{format(net)}</TableCell>
                  </TableRow>
                </TableFooter>
              </Table>
            )
          })
        )}
      </CardContent>
    </Card>
  )
}
