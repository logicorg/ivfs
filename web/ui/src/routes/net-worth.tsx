import { useQuery } from "@tanstack/react-query"
import { createFileRoute, Link } from "@tanstack/react-router"
import { Download } from "lucide-react"
import { Area, ComposedChart, Line, ReferenceArea, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"

import type { Currency, ProviderError } from "@/api"
import { listCurrenciesOptions, retrieveUserSettingsOptions } from "@/api/@tanstack/react-query.gen"
import { ProblemMessages } from "@/components/form-fields"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useNetWorth } from "@/hooks/use-net-worth"
import { useProviderChoices } from "@/hooks/use-provider-choices"
import { formatDate, localDate } from "@/lib/dates"
import { formatAmount } from "@/lib/money"
import { netWorthCsv, netWorthCsvName } from "@/lib/net-worth-csv"
import { flaggedStretches, monthsBefore, type NetWorthPoint } from "@/lib/net-worth-series"

// NetWorthSearch is the range and the reporting currency, kept in the URL so
// a chart can be bookmarked; the last year and the default currency if
// omitted.
type NetWorthSearch = { from?: string; to?: string; currencyId?: number }

const isDate = (value: unknown): value is string => typeof value === "string" && /^\d{4}-\d{2}-\d{2}$/.test(value)

export const Route = createFileRoute("/net-worth")({
  validateSearch: (search: Record<string, unknown>): NetWorthSearch => {
    const currencyId = Number(search.currencyId)
    return {
      ...(isDate(search.from) ? { from: search.from } : {}),
      ...(isDate(search.to) ? { to: search.to } : {}),
      ...(Number.isInteger(currencyId) && currencyId > 0 ? { currencyId } : {}),
    }
  },
  component: NetWorthScreen,
})

// The ranges a click away, in months back from the end.
const presets = [
  { label: "3M", months: 3 },
  { label: "1Y", months: 12 },
  { label: "5Y", months: 60 },
] as const

function NetWorthScreen() {
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const settings = useQuery(retrieveUserSettingsOptions())
  const currencies = useQuery(listCurrenciesOptions())

  const to = search.to ?? localDate(new Date())
  const from = search.from ?? monthsBefore(to, 12)
  const currencyId = search.currencyId ?? settings.data?.defaultCurrencyId
  const show = (next: NetWorthSearch) => navigate({ search: (previous) => ({ ...previous, ...next }) })

  return (
    <div className="flex max-w-5xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">Net worth</h1>

      <div className="flex flex-wrap items-end gap-4">
        <div className="flex flex-col gap-2">
          <Label htmlFor="net-worth-from">From</Label>
          <Input
            id="net-worth-from"
            type="date"
            value={from}
            max={to}
            onChange={(event) => isDate(event.target.value) && show({ from: event.target.value })}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="net-worth-to">To</Label>
          <Input
            id="net-worth-to"
            type="date"
            value={to}
            min={from}
            onChange={(event) => isDate(event.target.value) && show({ to: event.target.value })}
          />
        </div>
        <div className="flex gap-1">
          {presets.map(({ label, months }) => (
            <Button key={label} variant="outline" onClick={() => show({ from: monthsBefore(to, months) })}>
              {label}
            </Button>
          ))}
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="net-worth-currency">Currency</Label>
          <Select
            value={currencyId === undefined ? "" : String(currencyId)}
            onValueChange={(value) => show({ currencyId: Number(value) })}
          >
            <SelectTrigger id="net-worth-currency" className="w-40">
              <SelectValue placeholder="Pick one" />
            </SelectTrigger>
            <SelectContent>
              {[...(currencies.data?.currencies ?? [])]
                .sort((a, b) => a.code.localeCompare(b.code))
                .map((currency) => (
                  <SelectItem key={currency.id} value={String(currency.id)}>
                    {currency.code}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {settings.isPending ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : settings.isError ? (
        <ProblemMessages error={settings.error} />
      ) : currencyId === undefined ? (
        <p className="text-muted-foreground">
          Pick the currency to show net worth in, or set a default in{" "}
          <Link to="/settings" className="underline">
            settings
          </Link>
          .
        </p>
      ) : from > to ? (
        <p className="text-muted-foreground">The range ends before it starts; move From or To.</p>
      ) : (
        <NetWorthReport from={from} to={to} currencyId={currencyId} />
      )}
    </div>
  )
}

function NetWorthReport({ from, to, currencyId }: { from: string; to: string; currencyId: number }) {
  const { chosen } = useProviderChoices()
  const netWorth = useNetWorth(from, to, currencyId, chosen)

  if (netWorth.status === "pending") {
    return <p className="text-muted-foreground">Loading…</p>
  }
  if (netWorth.status === "error") {
    return <ProblemMessages error={netWorth.error} />
  }
  const { points, currency, providerErrors } = netWorth
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Over time</CardTitle>
          <CardDescription>
            In {currency.code}, at each date's rates. Shaded dates leave out a holding without a value or money
            without a rate; the table lists what.
          </CardDescription>
          <CardAction>
            <Button
              variant="outline"
              onClick={() => download(netWorthCsv(points, currency.code), netWorthCsvName(currency.code, from, to))}
            >
              <Download />
              CSV
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          <NetWorthChart points={points} currency={currency} />
          <ProviderErrors errors={providerErrors} />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>By date</CardTitle>
        </CardHeader>
        <CardContent>
          <NetWorthTable points={points} currency={currency} />
        </CardContent>
      </Card>
    </>
  )
}

// day shows a chart date, a UTC day, as a date: in the reader's own zone it
// could be the day before.
const day = (date: string) => formatDate(date, false, undefined, "UTC")

// The series, drawn as stacked areas: what is held above zero, what is owed
// below it, and net worth as the line between.
const series = [
  { key: "balances", label: "Balances", color: "var(--viz-balances)" },
  { key: "investments", label: "Investments", color: "var(--viz-investments)" },
  { key: "liabilities", label: "Liabilities", color: "var(--viz-liabilities)" },
] as const

function NetWorthChart({ points, currency }: { points: NetWorthPoint[]; currency: Currency }) {
  // Numbers only place the marks; every figure shown is formatted from the
  // exact strings.
  const data = points.map((point) => ({
    point,
    date: point.date,
    balances: Number(point.balances),
    investments: Number(point.investments),
    liabilities: -Number(point.liabilities),
    netWorth: Number(point.netWorth),
  }))
  const compact = new Intl.NumberFormat(undefined, { notation: "compact", maximumFractionDigits: 1 })

  return (
    <div className="flex flex-col gap-3">
      <ul className="flex flex-wrap gap-4 text-sm text-muted-foreground">
        {series.map(({ key, label, color }) => (
          <li key={key} className="flex items-center gap-2">
            <span className="size-3 rounded-sm" style={{ background: color }} />
            {label}
          </li>
        ))}
        <li className="flex items-center gap-2">
          <span className="h-0.5 w-4 bg-foreground" />
          Net worth
        </li>
      </ul>
      <div className="h-80">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={data} stackOffset="sign" margin={{ top: 8, right: 8, bottom: 0, left: 8 }}>
            {flaggedStretches(points).map(({ from, to }) => (
              <ReferenceArea key={from} x1={from} x2={to} fill="var(--muted-foreground)" fillOpacity={0.12} ifOverflow="hidden" />
            ))}
            <XAxis
              dataKey="date"
              tickFormatter={day}
              tick={{ fill: "var(--muted-foreground)", fontSize: 12 }}
              stroke="var(--border)"
              minTickGap={32}
            />
            <YAxis
              tickFormatter={(value: number) => compact.format(value)}
              tick={{ fill: "var(--muted-foreground)", fontSize: 12 }}
              stroke="var(--border)"
              width={56}
            />
            <Tooltip
              content={({ active, payload }) => {
                const point = active ? (payload?.[0]?.payload as { point: NetWorthPoint } | undefined)?.point : undefined
                return point ? <PointTooltip point={point} currency={currency} /> : null
              }}
              cursor={{ stroke: "var(--muted-foreground)", strokeDasharray: "3 3" }}
            />
            {series.map(({ key, color }) => (
              <Area
                key={key}
                dataKey={key}
                stackId="net-worth"
                type="linear"
                stroke={color}
                strokeWidth={2}
                fill={color}
                fillOpacity={0.15}
                isAnimationActive={false}
              />
            ))}
            <Line
              dataKey="netWorth"
              type="linear"
              stroke="var(--foreground)"
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
            />
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}

function PointTooltip({ point, currency }: { point: NetWorthPoint; currency: Currency }) {
  const format = (amount: string) => formatAmount(amount, currency.code, currency.precision)
  return (
    <div className="rounded-md border bg-popover px-3 py-2 text-sm text-popover-foreground shadow-md">
      <p className="font-medium">{day(point.date)}</p>
      <dl className="grid grid-cols-[auto_auto] gap-x-4 tabular-nums">
        {series.map(({ key, label, color }) => (
          <div key={key} className="contents">
            <dt className="flex items-center gap-2 text-muted-foreground">
              <span className="size-2 rounded-sm" style={{ background: color }} />
              {label}
            </dt>
            <dd className="text-right">{format(point[key])}</dd>
          </div>
        ))}
        <dt className="font-medium">Net worth</dt>
        <dd className="text-right font-medium">{format(point.netWorth)}</dd>
      </dl>
      {point.flags.length > 0 && <p className="mt-1 text-muted-foreground">Leaves out: {point.flags.join("; ")}</p>}
    </div>
  )
}

function NetWorthTable({ points, currency }: { points: NetWorthPoint[]; currency: Currency }) {
  const format = (amount: string) => formatAmount(amount, currency.code, currency.precision)
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Date</TableHead>
          <TableHead className="text-right">Balances</TableHead>
          <TableHead className="text-right">Investments</TableHead>
          <TableHead className="text-right">Liabilities</TableHead>
          <TableHead className="text-right">Net worth</TableHead>
          <TableHead>Leaves out</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {[...points].reverse().map((point) => (
          <TableRow key={point.date}>
            <TableCell>{day(point.date)}</TableCell>
            <TableCell className="text-right tabular-nums">{format(point.balances)}</TableCell>
            <TableCell className="text-right tabular-nums">{format(point.investments)}</TableCell>
            <TableCell className="text-right tabular-nums">{format(point.liabilities)}</TableCell>
            <TableCell className="text-right font-medium tabular-nums">{format(point.netWorth)}</TableCell>
            <TableCell className="text-muted-foreground">{point.flags.join("; ")}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

// ProviderErrors names the providers that did not answer, once each, since
// their silence can be why a date is flagged.
function ProviderErrors({ errors }: { errors: ProviderError[] }) {
  const unique = [...new Map(errors.map((error) => [`${error.provider}: ${error.detail}`, error])).keys()]
  if (unique.length === 0) {
    return null
  }
  return <p className="mt-3 text-sm text-muted-foreground">Did not answer: {unique.join("; ")}</p>
}

// download hands the browser text to save as a file.
function download(text: string, name: string) {
  const url = URL.createObjectURL(new Blob([text], { type: "text/csv;charset=utf-8" }))
  const link = document.createElement("a")
  link.href = url
  link.download = name
  link.click()
  // Revoked a moment later: Safari can still be reading it when click returns.
  setTimeout(() => URL.revokeObjectURL(url), 1_000)
}
