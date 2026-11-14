import { useInfiniteQuery, useQuery } from "@tanstack/react-query"
import { createFileRoute, Link } from "@tanstack/react-router"
import { ArrowLeft } from "lucide-react"

import {
  listAssetsOptions,
  listCurrenciesOptions,
  listFinancialAccountsOptions,
  listHoldingMovementsInfiniteOptions,
  listHoldingsOptions,
} from "@/api/@tanstack/react-query.gen"
import { ProblemMessages } from "@/components/form-fields"
import { JournalEntrySheet } from "@/components/journal-entry-sheet"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { formatDate } from "@/lib/dates"
import { formatUnits, holdingRows, type HoldingRow } from "@/lib/holdings"
import { formatAmount } from "@/lib/money"
import { movementRows } from "@/lib/movements"

// pageSize is how many movements each "Show later movements" loads.
const pageSize = 100

// HoldingSearch is the entry open in the sheet, kept in the URL.
type HoldingSearch = { entry?: number }

export const Route = createFileRoute("/holdings/$financialAccountId/$assetId")({
  validateSearch: (search: Record<string, unknown>): HoldingSearch => {
    const entry = Number(search.entry)
    return Number.isInteger(entry) && entry > 0 ? { entry } : {}
  },
  component: HoldingHistory,
})

// HoldingHistory is an asset held in a financial account, with every movement
// of its units. A movement's entry opens in the sheet, which deletes it, and
// the movement goes with it (ADR 0011).
function HoldingHistory() {
  const params = Route.useParams()
  const financialAccountId = Number(params.financialAccountId)
  const assetId = Number(params.assetId)
  const { entry } = Route.useSearch()
  const navigate = Route.useNavigate()
  const holdings = useQuery(listHoldingsOptions({ query: { financialAccountId, assetId } }))
  const accounts = useQuery(listFinancialAccountsOptions({ query: { includeClosed: true } }))
  const assets = useQuery(listAssetsOptions())
  const currencies = useQuery(listCurrenciesOptions())

  const queries = [holdings, accounts, assets, currencies]
  const failed = queries.find((query) => query.isError)
  // A holding with no movements left, as after deleting its only entry, is
  // still shown, as none held.
  const [holding] = holdingRows(
    holdings.data?.holdings.length
      ? holdings.data.holdings
      : [{ financialAccountId, assetId, units: "0", costBasis: [], unknownCostBasis: false }],
    accounts.data?.financialAccounts ?? [],
    assets.data?.assets ?? [],
    currencies.data?.currencies ?? [],
    true,
  )

  return (
    <div className="flex max-w-4xl flex-col gap-6">
      <Button variant="ghost" size="sm" className="self-start" asChild>
        <Link to="/investments">
          <ArrowLeft /> Investments
        </Link>
      </Button>

      {failed ? (
        <ProblemMessages error={failed.error} />
      ) : queries.some((query) => query.isPending) ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : (
        <>
          <HoldingHeader holding={holding} />
          <Movements
            financialAccountId={financialAccountId}
            assetId={assetId}
            onSelect={(entryId) => navigate({ search: { entry: entryId }, replace: entry !== undefined })}
          />
        </>
      )}

      <JournalEntrySheet
        entryId={entry}
        onSelect={(entryId) => navigate({ search: entryId === undefined ? {} : { entry: entryId } })}
      />
    </div>
  )
}

function HoldingHeader({ holding }: { holding: HoldingRow }) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold">{holding.assetName}</h1>
        <div className="flex items-center gap-2 text-muted-foreground">
          {holding.ticker && <Badge variant="outline">{holding.ticker}</Badge>}
          {holding.financialAccountName}
        </div>
      </div>
      <div className="text-right">
        <div className="text-sm text-muted-foreground">Units</div>
        <div className="text-2xl font-semibold tabular-nums">{formatUnits(holding.units)}</div>
        {holding.costBasis.map((cost) => (
          <div key={cost.currencyCode} className="text-sm text-muted-foreground tabular-nums">
            {formatAmount(cost.amount, cost.currencyCode, cost.precision)} cost
          </div>
        ))}
        {holding.unknownCostBasis && <Badge variant="outline">partly unknown</Badge>}
      </div>
    </div>
  )
}

// Movements lists the holding's movements, oldest first and a page at a time,
// as the units held can only be counted forwards.
function Movements({
  financialAccountId,
  assetId,
  onSelect,
}: {
  financialAccountId: number
  assetId: number
  onSelect: (entryId: number) => void
}) {
  const currencies = useQuery(listCurrenciesOptions())
  const movements = useInfiniteQuery({
    ...listHoldingMovementsInfiniteOptions({ query: { financialAccountId, assetId, limit: pageSize } }),
    // The first page has no after; each next page is after the previous one.
    initialPageParam: {},
    getNextPageParam: (page) => page.next,
  })

  if (movements.isPending) {
    return <p className="text-muted-foreground">Loading…</p>
  }
  if (movements.isError) {
    return <ProblemMessages error={movements.error} />
  }

  const rows = movementRows(
    movements.data.pages.flatMap((page) => page.movements),
    currencies.data?.currencies ?? [],
  )
  if (rows.length === 0) {
    return <p className="text-muted-foreground">No movements yet.</p>
  }

  return (
    <div className="flex flex-col gap-3">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-32">Date</TableHead>
            <TableHead>Description</TableHead>
            <TableHead className="text-right">Units</TableHead>
            <TableHead className="text-right">Cost basis</TableHead>
            <TableHead className="text-right">Held after</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => {
            // Only a trade's or a vest's movement has an entry to open.
            const entryId = row.journalEntryId
            return (
              <TableRow
                key={row.id}
                className={entryId === undefined ? undefined : "cursor-pointer"}
                onClick={entryId === undefined ? undefined : () => onSelect(entryId)}
              >
                <TableCell className="text-muted-foreground">{formatDate(row.occurredAt)}</TableCell>
                <TableCell>
                  <span className="flex items-center gap-2">
                    {entryId === undefined ? (
                      (row.description ?? <span className="text-muted-foreground">Movement {row.id}</span>)
                    ) : (
                      <button
                        type="button"
                        className="text-left hover:underline"
                        onClick={(event) => {
                          event.stopPropagation()
                          onSelect(entryId)
                        }}
                      >
                        {row.description ?? <span className="text-muted-foreground">Journal entry {entryId}</span>}
                      </button>
                    )}
                    <Badge variant="outline">{row.kind}</Badge>
                  </span>
                </TableCell>
                <TableCell className="text-right tabular-nums">{formatUnits(row.unitsDelta)}</TableCell>
                <TableCell className="text-right tabular-nums">
                  {row.costBasis ? (
                    formatAmount(row.costBasis.amount, row.costBasis.currencyCode, row.costBasis.precision)
                  ) : (
                    <span className="text-muted-foreground">—</span>
                  )}
                </TableCell>
                <TableCell className="text-right tabular-nums">{formatUnits(row.unitsAfter)}</TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
      {movements.hasNextPage && (
        <Button
          variant="outline"
          className="self-center"
          disabled={movements.isFetchingNextPage}
          onClick={() => void movements.fetchNextPage()}
        >
          {movements.isFetchingNextPage ? "Loading…" : "Show later movements"}
        </Button>
      )}
    </div>
  )
}
