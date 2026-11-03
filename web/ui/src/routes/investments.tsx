import { useState } from "react"
import { useQueries, useQuery, type UseQueryResult } from "@tanstack/react-query"
import { createFileRoute, Link } from "@tanstack/react-router"

import type { AssetValues, Currency, ErrorModel } from "@/api"
import {
  listAssetsOptions,
  listAssetValuesOptions,
  listCurrenciesOptions,
  listFinancialAccountsOptions,
  listHoldingsOptions,
} from "@/api/@tanstack/react-query.gen"
import { BuyDialog } from "@/components/buy-dialog"
import { ProblemMessages } from "@/components/form-fields"
import { SellDialog } from "@/components/sell-dialog"
import { VestDialog } from "@/components/vest-dialog"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useProviderChoices, type Choices } from "@/hooks/use-provider-choices"
import { marketValue } from "@/lib/asset-values"
import { formatDate, localDate } from "@/lib/dates"
import { formatUnits, holdingRows, type HoldingRow, type Money } from "@/lib/holdings"
import { formatAmount } from "@/lib/money"
import { problemMessages } from "@/lib/problem"

// InvestmentsSearch is the screen's filter, kept in the URL.
type InvestmentsSearch = { includeSold?: boolean }

export const Route = createFileRoute("/investments")({
  validateSearch: (search: Record<string, unknown>): InvestmentsSearch =>
    search.includeSold === true || search.includeSold === "true" ? { includeSold: true } : {},
  component: Investments,
})

function Investments() {
  const { includeSold } = Route.useSearch()
  const navigate = Route.useNavigate()
  const holdings = useQuery(listHoldingsOptions())
  // Closed accounts too: units can still sit in one, and its holding needs a
  // name.
  const accounts = useQuery(listFinancialAccountsOptions({ query: { includeClosed: true } }))
  const assets = useQuery(listAssetsOptions())
  const currencies = useQuery(listCurrenciesOptions())
  const [buying, setBuying] = useState(false)
  const [selling, setSelling] = useState(false)
  const [vesting, setVesting] = useState(false)

  const choices = useProviderChoices()

  const queries = [holdings, accounts, assets, currencies]
  const failed = queries.find((query) => query.isError)
  const rows = holdingRows(
    holdings.data?.holdings ?? [],
    accounts.data?.financialAccounts ?? [],
    assets.data?.assets ?? [],
    currencies.data?.currencies ?? [],
    includeSold,
  )

  return (
    <div className="flex max-w-4xl flex-col gap-6">
      <div className="flex items-center">
        <h1 className="text-2xl font-semibold">Investments</h1>
        <div className="ml-auto flex gap-2">
          <Button onClick={() => setBuying(true)}>Buy</Button>
          <Button variant="outline" onClick={() => setSelling(true)}>
            Sell
          </Button>
          <Button variant="outline" onClick={() => setVesting(true)}>
            Vest
          </Button>
        </div>
        <BuyDialog open={buying} onOpenChange={setBuying} />
        <SellDialog open={selling} onOpenChange={setSelling} />
        <VestDialog open={vesting} onOpenChange={setVesting} />
      </div>

      <div className="flex items-center gap-2">
        <Checkbox
          id="include-sold"
          checked={includeSold ?? false}
          onCheckedChange={(checked) => navigate({ search: checked === true ? { includeSold: true } : {} })}
        />
        <Label htmlFor="include-sold">Show holdings sold to zero</Label>
      </div>

      {failed ? (
        <ProblemMessages error={failed.error} />
      ) : queries.some((query) => query.isPending) ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : rows.length === 0 ? (
        <p className="text-muted-foreground">No holdings yet.</p>
      ) : (
        <HoldingTable rows={rows} currencies={currencies.data?.currencies ?? []} choices={choices} />
      )}
    </div>
  )
}

function HoldingTable({
  rows,
  currencies,
  choices,
}: {
  rows: HoldingRow[]
  currencies: Currency[]
  choices: Choices
}) {
  // Today's value, asked once per asset however many accounts hold it.
  const today = localDate(new Date())
  const assetIds = [...new Set(rows.map((row) => row.assetId))]
  const values = useQueries({
    queries: assetIds.map((id) => listAssetValuesOptions({ path: { id }, query: { from: today, to: today } })),
  })
  const valuesByAsset = new Map(assetIds.map((id, i) => [id, values[i]]))

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Account</TableHead>
          <TableHead>Asset</TableHead>
          <TableHead className="text-right">Units</TableHead>
          <TableHead className="text-right">Cost basis</TableHead>
          <TableHead className="text-right">Average cost</TableHead>
          <TableHead className="text-right">Market value</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={`${row.financialAccountId}-${row.assetId}`}>
            <TableCell>{row.financialAccountName}</TableCell>
            <TableCell>
              <Link
                to="/holdings/$financialAccountId/$assetId"
                params={{ financialAccountId: String(row.financialAccountId), assetId: String(row.assetId) }}
                className="flex items-center gap-2 hover:underline"
              >
                {row.assetName}
                {row.ticker && <span className="text-muted-foreground">{row.ticker}</span>}
              </Link>
            </TableCell>
            <TableCell className="text-right tabular-nums">{formatUnits(row.units)}</TableCell>
            <TableCell className="text-right tabular-nums">
              <span className="flex flex-col items-end">
                {row.costBasis.map((cost) => (
                  <span key={cost.currencyCode}>{format(cost)}</span>
                ))}
                {row.unknownCostBasis && <Badge variant="outline">partly unknown</Badge>}
              </span>
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {row.averageCost ? format(row.averageCost) : <span className="text-muted-foreground">—</span>}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              <MarketValueCell
                units={row.units}
                values={valuesByAsset.get(row.assetId)}
                chosen={choices.chosen[row.assetId]}
                onChoose={(provider) => choices.choose(row.assetId, provider)}
                currencies={currencies}
              />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

// format shows a cost at its currency's precision, so an average cost is
// rounded for reading; the exact one stays in the ledger.
function format({ amount, currencyCode, precision }: Money): string {
  return formatAmount(amount, currencyCode, precision)
}

// MarketValueCell shows what the holding is worth today by the chosen
// provider, with a pick of the providers that have a value and why the
// others do not.
function MarketValueCell({
  units,
  values,
  chosen,
  onChoose,
  currencies,
}: {
  units: string
  values?: UseQueryResult<AssetValues, ErrorModel>
  chosen?: string
  onChoose: (provider: string) => void
  currencies: Currency[]
}) {
  if (!values || values.isPending) {
    return <span className="text-muted-foreground">…</span>
  }
  if (values.isError) {
    return (
      <Tooltip>
        <TooltipTrigger className="text-muted-foreground">unavailable</TooltipTrigger>
        <TooltipContent>{problemMessages(values.error).join(" ")}</TooltipContent>
      </Tooltip>
    )
  }
  const worth = marketValue(units, values.data, chosen, currencies)
  const errors = values.data.errors
  return (
    <span className="flex flex-col items-end gap-1">
      {worth ? (
        <span title={`${format(worth.price)} a unit, observed ${formatDate(worth.observedAt, true)}`}>
          {format(worth.value)}
        </span>
      ) : (
        <span className="text-muted-foreground">—</span>
      )}
      <span className="flex items-center gap-1">
        {errors.length > 0 && (
          <Tooltip>
            <TooltipTrigger>
              <Badge variant="outline">{errors.length} unavailable</Badge>
            </TooltipTrigger>
            <TooltipContent>
              {errors.map((error) => (
                <p key={error.provider}>
                  {error.provider}: {error.detail}
                </p>
              ))}
            </TooltipContent>
          </Tooltip>
        )}
        {worth && worth.providers.length > 1 ? (
          <Select value={worth.provider} onValueChange={onChoose}>
            <SelectTrigger size="sm" aria-label="Value provider">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {worth.providers.map((provider) => (
                <SelectItem key={provider} value={provider}>
                  {provider}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          worth && <span className="text-xs text-muted-foreground">{worth.provider}</span>
        )}
      </span>
    </span>
  )
}
